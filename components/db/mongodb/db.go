// Copyright (C) 2017, Beijing Bochen Technology Co.,Ltd.  All rights reserved.
//
// This file is part of L0
//
// The L0 is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The L0 is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
//
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with this program.  If not, see <http://www.gnu.org/licenses/>.

package mongodb

import (
	"encoding/json"
	"errors"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/bocheninc/base/log"
	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
)

type Config struct {
	Hosts  []string
	DBName string
}

type Mdb struct {
	session  *mgo.Session
	database *mgo.Database
	cols     map[string]struct{}
	sync.Mutex
	cfg Config
}

var db *Mdb
var once sync.Once

func DefaultConfig() *Config {
	return &Config{
		Hosts:  []string{"127.0.0.1"},
		DBName: "test",
	}
}

func NewMdb(cfg *Config) (*Mdb, error) {
	var err error
	log.Infof("mdb cfg: %+v", cfg)
	once.Do(func() {
		if cfg == nil {
			panic("if nvp, please support mongodb")
		}

		db = &Mdb{
			cfg:  Config{Hosts: cfg.Hosts, DBName: cfg.DBName},
			cols: make(map[string]struct{}),
		}
		err = db.init()
	})

	return db, err
}

func MongDB() *Mdb {
	return db
}

func (db *Mdb) init() error {
	var err error
	dialInfo := &mgo.DialInfo{
		Addrs:     db.cfg.Hosts,
		Direct:    false,
		Timeout:   time.Second * 10,
		PoolLimit: 4096,
	}

	db.session, err = mgo.DialWithInfo(dialInfo)

	if err != nil {
		log.Println(err.Error())
		return err
	}
	db.session.SetMode(mgo.Monotonic, true)
	db.database = db.session.DB(db.cfg.DBName)

	return nil
}

func (db *Mdb) Coll(col string) *mgo.Collection {
	if !db.HaveCollection(col) {
		log.Errorf("db collection not register: %+v", col)
		return nil
	}
	return db.database.C(col)
}

func (db *Mdb) RegisterCollection(col string) {
	db.Lock()
	defer db.Unlock()
	db.cols[col] = struct{}{}
}

func (db *Mdb) UnRegisterCollection(col string) {
	db.Lock()
	defer db.Unlock()
	delete(db.cols, col)
}

func (db *Mdb) HaveCollection(col string) bool {
	db.Lock()
	defer db.Unlock()
	_, ok := db.cols[col]
	return ok
}

func (db *Mdb) Query(key string) ([]byte, error) {
	params, err := db.checkFormat(key)
	if err != nil {
		return nil, err
	}
	var (
		col     string
		query   *mgo.Query
		results []interface{}
		result  interface{}
	)

	for _, v := range params {
		for method, param := range v {
			if param == "collection" {
				col = method
				continue
			}
			if method == "findOne" {
				var m bson.M
				var err error
				if param != nil {
					m, err = convertBson(param)
					if err != nil {
						return nil, err
					}
				}
				if err := db.Coll(col).Find(m).One(&result); err != nil {
					return nil, err
				}
				data, err := json.Marshal(result)
				if err != nil {
					return nil, err
				}
				return data, nil
			}

			query, err = db.execQuery(col, method, param, query)
			if err != nil {
				return nil, err
			}
		}
	}
	if err := query.All(&results); err != nil {
		return nil, err
	}
	data, err := json.Marshal(results)
	if err != nil {
		return nil, err
	}
	return data, nil

}

func (db *Mdb) execQuery(col, method string, param interface{}, query *mgo.Query) (*mgo.Query, error) {
	var m bson.M
	var err error

	switch method {
	case "find":
		if param != nil {
			m, err = convertBson(param)
		}
		query = db.Coll(col).Find(m)
	case "limit":
		limit, ok := param.(float64)
		if !ok {
			return nil, errors.New("limit number must be float64")
		}
		query = query.Limit(int(limit))
	case "skip":
		skip, ok := param.(float64)
		if !ok {
			return nil, errors.New("skip number must be float64")
		}
		query = query.Skip(int(skip))
	case "sort":
		var fields []string
		for k, v := range param.(map[string]interface{}) {
			specification, ok := v.(float64)
			if !ok || specification < -1 || specification > 1 {
				return nil, errors.New("bad sort specification number must be float64 type , equal 1 or -1")
			}
			if specification == -1 {
				k = "-" + k
			}
			fields = append(fields, k)
		}
		query = query.Sort(fields...)
	default:
		return nil, errors.New("not suppot sql method: " + method)
	}

	if err != nil {
		return nil, err
	}
	return query, nil
}

func (db *Mdb) checkFormat(key string) ([]map[string]interface{}, error) {
	var params []map[string]interface{}

	//check key if not nil
	if len(key) == 0 {
		return nil, errors.New("query key must not be nil ")
	}

	//delete space
	key = strings.TrimSpace(key)

	results, err := parseMethodAndParams(key)
	if err != nil {
		return nil, err
	}

	params = append(params, results...)

	return params, nil
}

func parseMethodAndParams(methodAndParams string) ([]map[string]interface{}, error) {
	var results []map[string]interface{}
	regMethod := regexp.MustCompile(`(\w+)(\(([\w:"\{\},\[\]\.\$ -]*)\))*`)
	methodParams := regMethod.FindAllStringSubmatch(methodAndParams, -1)
	for k, v := range methodParams {
		if len(v) != 4 {
			return nil, errors.New("not support query key")
		}

		result := make(map[string]interface{})
		switch k {
		case 0:
			//check first params is db
			if v[1] != "db" {
				return nil, errors.New("query key first param must be 'db'")
			}
		case 1:
			//check collection
			if !db.HaveCollection(v[0]) {
				return nil, errors.New("collection: " + v[0] + " is not exist")
			}
			result[v[1]] = "collection"
		case 2:
			if v[1] != "find" && v[1] != "findOne" {
				return nil, errors.New("query key  must be find method")
			}
			fallthrough
		default:
			var m interface{}
			if len(v[3]) != 0 {
				if err := json.Unmarshal([]byte(parseParam(v[3])), &m); err != nil {
					return nil, err
				}
			}
			result[v[1]] = m
		}
		results = append(results, result)
	}
	return results, nil
}

func parseParam(param string) string {
	var result string
	index := strings.IndexAny(param, "$")
	if index != -1 {
		result = param[:index] + `"`
		colonIndex := strings.IndexAny(param[index:], ":")
		if colonIndex != -1 {
			result = result + param[index:][:colonIndex] + `"` + parseParam(param[index:][colonIndex:])
		}
	} else {
		result = param
	}
	return result
}

func convertBson(src interface{}) (bson.M, error) {
	data, err := bson.Marshal(src)
	if err != nil {
		return nil, err
	}
	m := bson.M{}
	if err := bson.Unmarshal(data, m); err != nil {
		return nil, err
	}
	return m, nil
}

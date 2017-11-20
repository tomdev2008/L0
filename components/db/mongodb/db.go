package mongodb

import (
	"encoding/json"
	"errors"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/bocheninc/L0/components/log"
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
		Timeout:   time.Second * 1,
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
		return nil, errors.New("not supot sql method: " + method)
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

	// check key is find method
	if !isFind(key) {
		return nil, errors.New("query key  must be find method")
	}

	paramsSlice := strings.Split(key, ".")

	//check first params is db
	if !isdb(paramsSlice[0]) {
		return nil, errors.New("query key first param must be 'db'")
	}

	//check collection
	if !db.HaveCollection(paramsSlice[1]) {
		return nil, errors.New("collection: " + paramsSlice[1] + " is not exist")
	}
	collectionParam := make(map[string]interface{})
	collectionParam[paramsSlice[1]] = "collection"
	params = append(params, collectionParam)

	methodAndParams := strings.Join(paramsSlice[2:], ".")

	results, err := parseMethodAndParams(methodAndParams)
	if err != nil {
		return nil, err
	}

	params = append(params, results...)

	return params, nil
}

func parseMethodAndParams(methodAndParams string) ([]map[string]interface{}, error) {
	var results []map[string]interface{}

	regMethod := regexp.MustCompile(`(\w+)\([\w:"\{\},\.\$ -]*\)`)
	methodParams := regMethod.FindAllString(methodAndParams, -1)

	for _, v := range methodParams {
		methodParamsSlice := strings.Split(v, "(")
		if len(methodParamsSlice) != 2 {
			return nil, errors.New("not support query key")
		}
		result := make(map[string]interface{})

		var m interface{}
		param := strings.Trim(methodParamsSlice[1], ")")
		if len(param) != 0 {
			if err := json.Unmarshal([]byte(parseParam(param)), &m); err != nil {
				return nil, err
			}
		}
		result[methodParamsSlice[0]] = m
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

//
//type MockPerson struct {
//	//Id      bson.ObjectId `bson:"_id,omitempty" json:"_id"`
//	Name    string
//	Country string
//	Age     int
//}
//
//type MP struct {
//	//Id      bson.ObjectId `bson:"_id,omitempty" json:"_id"`
//	Name    []byte
//	Country string
//	Age     int
//}
//
////Bulk Insert
//func testBulkInsert(col *mgo.Collection) {
//	fmt.Println("Test Bulk Insert into MongoDB")
//	bulk := col.Bulk()
//
//	var contentArray []interface{}
//	contentArray = append(contentArray, &MockPerson{
//		//Id:      bson.ObjectId("5a0d8195b1fd"),
//		Name:    "aaaaaaaaaaaaaaaa",
//		Age:     1,
//		Country: "USA",
//	})
//
//	contentArray = append(contentArray, &MockPerson{
//		Name:    "this-is-good-content84344",
//		Age:     10,
//		Country: "USA2",
//	})
//
//	//contentArray = contentArray
//	//bulk.Insert(contentArray...)
//	//bulk.Remove(bson.M{"_id": bson.ObjectId("5a0d8195b1fd")})
//	//bulk.Upsert(bson.M{"_id": "gdfauddfaffafassss"}, &MP{Name: []byte("xxfgsx"), Country: "china", Age: 100})
//
//	data, _ := bson.MarshalJSON(MP{Name: []byte("xxfgsx"), Country: "china", Age: 100})
//	var value interface{}
//	bson.Unmarshal(data, &value)
//	//bulk.Insert(bson.M{"_id": "gdxx666666xxssss"}, bson.Binary{Kind: '0', Data: []byte("000")})
//	//col.Insert(&bson.Binary{Kind: '0', Data: []byte("000000000000000000")})
//	//data, err := bson.Marshal(MP{Name: []byte("0000000000000"), Country: "china", Age: 100})
//	//if err != nil {
//	//	panic(err)
//	//}
//	//bulk.Upsert(bson.M{"_id": "gdfxxxxssss"}, data)
//	bulk.Insert(MP{Name: []byte("0000000000000"), Country: "china", Age: 100})
//	_, err := bulk.Run()
//	if err != nil {
//		panic(err)
//	}
//}
//
//type MapMP struct {
//	Amounts map[int]*big.Int
//	Nonce   uint32
//	rw      sync.RWMutex
//}
//
//func testInsertMap(col *mgo.Collection) {
//	bulk := col.Bulk()
//
//	var value interface{}
//	data, _ := json.Marshal(MapMP{Amounts: map[int]*big.Int{1: big.NewInt(3344), 11: big.NewInt(20000)}, Nonce: 1})
//	json.Unmarshal(data, &value)
//	fmt.Println("==>>>>>>>>>>>>>>", value)
//	//_, err := col.UpsertId("f222222rtdfaf4wetw222222222f", value)
//	//_, err := col.UpsertId("f222222222222222f", value)
//	bulk.Upsert(bson.M{"_id": "45720turfjdrrt0748fap"}, value)
//	_, err := bulk.Run()
//	if err != nil {
//		panic(err)
//	}
//
//	//var value interface{}
//	//data, _ := json.Marshal(MapMP{Amounts: map[string]*big.Int{"1": big.NewInt(100), "11": big.NewInt(200)}, Nonce: 1})
//	//json.Unmarshal(data, &value)
//	//fmt.Println("==>>>>>>>>>>>>>>", value)
//}
//
//func test() {
//	db, err := NewMdb(DefaultConfig())
//	if err != nil {
//		panic(err)
//	}
//	db.RegisterCollection("person")
//	db.RegisterCollection("transaction")
//	db.RegisterCollection("block")
//	db.RegisterCollection("balance")
//
//	mp := MockPerson{Name: "wang6", Country: "china9", Age: 19}
//	jmp, _ := json.Marshal(mp)
//	var jump interface{}
//	json.Unmarshal(jmp, &jump)
//	//jump.(map[string]interface{})["_id"] = "guess"
//
//	//err = db.Coll("person").Update(map[string]string{"Name": "wang"}, jump)
//	//err = db.Coll("person").UpdateId("guess", jump)
//	//_, err = db.Coll("person").UpsertId("sleep", jump)
//	//db.Coll("person").Insert(bson.M{"name": "dddd"})
//	//testBulkInsert(db.Coll("person"))
//	testInsertMap(db.Coll("person"))
//	var tx interface{}
//	//bson.MarshalJSON(MP{Name: []byte("xxfgsx"), Country: "china", Age: 100})
//	//search, err := bson.Marshal([]byte("0000000000000"))
//	if err != nil {
//		panic(err)
//	}
//
//	//fmt.Println(reflect.TypeOf(search), search)
//	//err = db.Coll("balance").Find(bson.M{"data.tochain": []byte{0}}).One(&tx)
//	db.Coll("balance").Find(bson.M{"nonce": 1}).One(&tx)
//	if err != nil {
//		fmt.Println("insert err: ", err)
//	}
//
//	fmt.Println(tx)
//
//	//db.Coll("person").Insert(map[string]string{"_id": "12345678", "hello": "world"})
//}

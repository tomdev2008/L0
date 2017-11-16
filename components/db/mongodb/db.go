package mongodb

import (
	"github.com/bocheninc/L0/components/log"
	"gopkg.in/mgo.v2"
	"sync"
	"time"
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
	once.Do(func() {
		if cfg == nil {
			panic("if nvp, please support mongodb")
		}

		db = &Mdb{
			cfg: Config{Hosts: cfg.Hosts, DBName: cfg.DBName},
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
		log.Errorf("db collection not register")
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

func test() {
	//db := NewMdb(Config{Hosts: []string{"127.0.0.1"}, DBName: "test"})
	db.RegisterCollection("transaction")
	db.Coll("person").Insert(map[string]string{"hello": "world"})
	db.Coll("bank").Insert(map[string]string{"hello": "world"})
}

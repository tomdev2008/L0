package mongodb

import (
	"encoding/json"
	"fmt"
	"github.com/bocheninc/L0/components/log"
	"gopkg.in/mgo.v2"
	//"gopkg.in/mgo.v2/bson"
	//"gopkg.in/mgo.v2/bson"
	"gopkg.in/mgo.v2/bson"
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

type MockPerson struct {
	Id      bson.ObjectId `bson:"_id,omitempty" json:"_id"`
	Name    string
	Country string
	Age     int
}

//Bulk Insert
func testBulkInsert(col *mgo.Collection) {
	fmt.Println("Test Bulk Insert into MongoDB")
	bulk := col.Bulk()

	var contentArray []interface{}
	contentArray = append(contentArray, &MockPerson{
		Id:      bson.ObjectId("5a0d8195b1fd"),
		Name:    "aaaaaaaaaaaaaaaa",
		Age:     1,
		Country: "USA",
	})

	contentArray = append(contentArray, &MockPerson{
		Name:    "this-is-good-content84344",
		Age:     10,
		Country: "USA2",
	})

	//contentArray = contentArray
	//bulk.Insert(contentArray...)
	bulk.Remove(bson.M{"_id": bson.ObjectId("5a0d8195b1fd")})
	_, err := bulk.Run()
	if err != nil {
		panic(err)
	}
}

func main() {
	db, err := NewMdb(DefaultConfig())
	if err != nil {
		panic(err)
	}
	db.RegisterCollection("person")

	mp := MockPerson{Name: "wang6", Country: "china9", Age: 19}
	jmp, _ := json.Marshal(mp)
	var jump interface{}
	json.Unmarshal(jmp, &jump)
	//jump.(map[string]interface{})["_id"] = "guess"

	//err = db.Coll("person").Update(map[string]string{"Name": "wang"}, jump)
	//err = db.Coll("person").UpdateId("guess", jump)
	//_, err = db.Coll("person").UpsertId("sleep", jump)
	//db.Coll("person").Insert(bson.M{"name": "dddd"})
	testBulkInsert(db.Coll("person"))
	if err != nil {
		fmt.Println("insert err: ", err)
	}
	//db.Coll("person").Insert(map[string]string{"_id": "12345678", "hello": "world"})
}

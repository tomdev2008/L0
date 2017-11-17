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

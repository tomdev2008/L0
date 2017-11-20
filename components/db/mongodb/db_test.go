package mongodb

import (
	"fmt"
	"testing"

	mgo "gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
)

type MockPerson struct {
	//Id      bson.ObjectId `bson:"_id,omitempty" json:"_id"`
	Name    string
	Country string
	Age     int
	Payload []byte
}

//Bulk Insert
func testBulkInsert(col *mgo.Collection) {
	fmt.Println("Test Bulk Insert into MongoDB")
	bulk := col.Bulk()

	var contentArray []interface{}
	contentArray = append(contentArray, &MockPerson{
		//Id:      bson.ObjectId("5a0d8195b1fd"),
		Name:    "aaaaaaaaaaaaaaaa",
		Age:     1,
		Country: "USA",
		Payload: []byte("1111"),
	})

	contentArray = append(contentArray, &MockPerson{
		Name:    "this-is-good-content84344",
		Age:     10,
		Country: "USA2",
		Payload: []byte("2222"),
	})

	contentArray = append(contentArray, &MockPerson{
		Name:    "bbbbbbbbbbbbbbbbb",
		Age:     9,
		Country: "USA1",
		Payload: []byte("3333"),
	})

	bulk.Insert(contentArray...)

	_, err := bulk.Run()
	if err != nil {
		panic(err)
	}
}

func remove(col *mgo.Collection) {

	if _, err := col.RemoveAll(bson.M{"age": 1}); err != nil {
		panic(err)
	}
	if _, err := col.RemoveAll(bson.M{"age": 9}); err != nil {
		panic(err)
	}
	if _, err := col.RemoveAll(bson.M{"age": 10}); err != nil {
		panic(err)
	}

}

func TestCheckFormat(t *testing.T) {
	db, err := NewMdb(DefaultConfig())
	if err != nil {
		t.Error(err)
	}

	db.RegisterCollection("person")
	db.RegisterCollection("transaction")
	db.RegisterCollection("block")
	db.RegisterCollection("balance")

	testBulkInsert(db.Coll("person"))

	//key := `db.person.findOne()`
	//key := `db.person.find()`
	//key := `db.person.find().limit(1)`
	//key := `db.person.find().skip(1)`
	//key := `db.person.find().skip(1).limit(1)`
	//key := `db.person.find().sort({"age":1})`
	//key := `db.person.find({"age":1})`
	//key := `db.person.find({"age":{$gt:1}})`

	key := `db.person.find({"age":{$lt:10,$gt:1}})`

	result, err := db.Query(key)
	if err != nil {
		t.Error("db query ", err)
	}
	t.Log("result: ", string(result))

	remove(db.Coll("person"))

}

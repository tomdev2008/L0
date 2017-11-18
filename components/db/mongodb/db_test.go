package mongodb

import (
	"encoding/json"
	"fmt"
	"testing"
)

func TestCheckFormat(t *testing.T) {
	db, _ := NewMdb(DefaultConfig())

	key := `db.col.find({"$or":[{"by":"菜鸟教程"},{"title": "MongoDB 教程"}]}).pretty(15)`
	params, err := db.checkFormat(key)
	if err != nil {
		t.Error(err)
	}

	fmt.Println("params: ", params)

	m1 := make(map[string]interface{})

	// m1["haha"] = "sdsa"
	// data, err := bson.Marshal(&params[0])
	// if err != nil {
	// 	t.Error(err)
	// }

	// fmt.Printf("%q", data)

	// m := bson.M{}
	// if err := bson.Unmarshal(data, m); err != nil {
	// 	t.Error(err)
	// }

	// fmt.Println("---m: ", m)

	if err := json.Unmarshal([]byte(`{$or:[{"by":"菜鸟教程"},{"title": "MongoDB 教程"}]}`), &m1); err != nil {
		t.Error(err)
	}

	fmt.Println("---m: ", m1)

}

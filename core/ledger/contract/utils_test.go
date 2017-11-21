package contract

import (
	"encoding/json"
	"fmt"
	"testing"
)

var (
	encBytes []byte
	err      error
)

func IsJson(src []byte) bool {
	var value interface{}
	return json.Unmarshal(src, &value) == nil
}

func TestConcrateStateJson(t *testing.T) {
	encData, err := ConcrateStateJson(DefaultAdminAddr)
	if err != nil {
		fmt.Println("Enc to json err: ", err)
	}
	encBytes = encData.Bytes()
}

func TestDoContractStateData(t *testing.T) {
	fmt.Println("enc: ", encBytes)
	decBytes, err := DoContractStateData(encBytes)
	if err != nil || decBytes == nil && err == nil {
		fmt.Println("Dec to origin data err: ", decBytes)
	}

	ok := IsJson(decBytes)
	fmt.Println("res: ", ok)
}

package state

import (
	"bytes"
	"encoding/json"
	"fmt"

	"github.com/bocheninc/L0/core/accounts"
	"github.com/bocheninc/base/log"
)

//Asset Attributes
type Asset struct {
	ID         uint32 `json:"id"`         // id
	Name       string `json:"name"`       // name
	Descr      string `json:"descr"`      // description
	Precision  uint64 `json:"precision"`  // divisible, precision
	Expiration uint32 `json:"expiration"` // expriation datetime

	Issuer accounts.Address `json:"issuer"` // issuer address
	Owner  accounts.Address `json:"owner"`  // owner address
}

//Update
func (asset *Asset) Update(jsonStr string) (*Asset, error) {
	if len(jsonStr) == 0 {
		return asset, nil
	}
	tAsset := &Asset{}
	if err := json.Unmarshal([]byte(jsonStr), tAsset); err != nil {
		return nil, fmt.Errorf("invalid json string for asset - %s", err)
	}

	var newVal map[string]interface{}
	json.Unmarshal([]byte(jsonStr), &newVal)

	oldJSONAStr, _ := json.Marshal(asset)
	var oldVal map[string]interface{}
	json.Unmarshal(oldJSONAStr, &oldVal)

	for k, val := range newVal {
		if _, ok := oldVal[k]; ok {
			oldVal[k] = val
		}
	}

	bts, _ := json.Marshal(oldVal)
	newAsset := &Asset{}
	json.Unmarshal(bts, newAsset)

	if asset.ID != newAsset.ID ||
		!bytes.Equal(asset.Issuer.Bytes(), newAsset.Issuer.Bytes()) ||
		!bytes.Equal(asset.Owner.Bytes(), newAsset.Owner.Bytes()) {

		log.Errorf("asset update failed, attribute mismatch, from %#v to %#v",
			asset, newAsset)
		return nil, fmt.Errorf("id, issuer, owner are readonly attribute, can't modified")
	}

	return newAsset, nil
}

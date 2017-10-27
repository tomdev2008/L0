package state

import (
	"encoding/json"
	"fmt"
	"math/big"

	"github.com/bocheninc/L0/core/accounts"
)

//AssetType
const (
	AssetUtilityToken uint32 = iota
	AssetToken
)

//Asset Attributes
type Asset struct {
	ID         uint32   `json:"id"`         // id
	Name       string   `json:"name"`       // name
	Descr      string   `json:"descr"`      // description
	Type       uint32   `json:"type"`       // type
	Amount     *big.Int `json:"amount"`     // used
	Available  *big.Int `json:"available"`  // unused
	Precision  uint64   `json:"precision"`  // divisible, precision
	Expiration uint32   `json:"expiration"` // expriation datetime
	// issuer > admin > owner > fee
	Issuer accounts.Address `json:"issuer"` // issuer address
	Admin  accounts.Address `json:"admin"`  // admin address
	Owner  accounts.Address `json:"owner"`  // owner address
	// Fee
	Fee        *big.Int         `json:"fee"`
	FeeAddress accounts.Address `json:"feeOwner"` //fee address
}

//Copy
func (asset *Asset) Update(jsonStr string) (*Asset, error) {
	tAsset := &Asset{}
	if err := json.Unmarshal([]byte(jsonStr), tAsset); err != nil {
		return nil, fmt.Errorf("invalid asset json string")
	}

	var newVal map[string]interface{}
	json.Unmarshal([]byte(jsonStr), &newVal)

	oldJSONAStr, _ := json.Marshal(asset)
	var oldVal map[string]interface{}
	json.Unmarshal(oldJSONAStr, &oldVal)

	for k := range oldVal {
		if val, ok := newVal[k]; ok {
			oldVal[k] = val
		}
	}

	bytes, _ := json.Marshal(oldVal)
	newAsset := &Asset{}
	json.Unmarshal(bytes, newAsset)

	if asset.ID != newAsset.ID {
		return nil, fmt.Errorf("id %d is readonly, not allowed to write", asset.ID)
	}

	return newAsset, nil
}

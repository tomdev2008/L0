package state

import "math/big"

type Balance struct {
	Amounts map[uint32]*big.Int
}

func NewBalance() *Balance {
	return &Balance{
		Amounts: make(map[uint32]*big.Int),
	}
}
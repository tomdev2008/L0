package lbft

import (
	"math/big"
	"time"

	"github.com/bocheninc/L0/components/crypto"
	"github.com/bocheninc/L0/core/accounts"
	"github.com/bocheninc/L0/core/coordinate"
	"github.com/bocheninc/L0/core/types"
	"github.com/bocheninc/base/log"
)

func (lbft *Lbft) testConsensus() {
	go func() {
		for {
			select {
			case <-lbft.outputTxsChan:
			}
		}
	}()

	var txs types.Transactions
	for i := 0; i < 100; i++ {
		priv1, _ := crypto.GenerateKey()
		addr1 := accounts.PublicKeyToAddress(*(priv1.Public()))
		priv2, _ := crypto.GenerateKey()
		addr2 := accounts.PublicKeyToAddress(*(priv2.Public()))
		tx := types.NewTransaction(
			coordinate.NewChainCoordinate([]byte{0}),
			coordinate.NewChainCoordinate([]byte{0}),
			types.TypeAtomic,
			uint32(time.Now().UnixNano()),
			addr1,
			addr2,
			uint32(0),
			big.NewInt(0),
			big.NewInt(0),
			uint32(time.Now().UnixNano()),
		)
		sig, _ := priv1.Sign(tx.SignHash().Bytes())
		tx.WithSignature(sig)
		txs = append(txs, tx)
	}

	go func() {
		i := 0
		start := time.Now()
		for {
			select {
			case <-lbft.testChan:
				if lbft.isPrimary() {
					if i == 0 {
						start = time.Now()
					}
					i++
					lbft.ProcessBatch(txs, func(int, types.Transactions) {
					})
					if i%50 == 0 {
						log.Debugf("testing ... txs: %d  speed: %s ", len(txs), time.Now().Sub(start))
						if len(txs) >= 10000 {
							txs = txs[:100]
						} else {
							txs = append(txs, txs...)
						}
						start = time.Now()
					}
				}
			}
		}
	}()
}

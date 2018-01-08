package notify

import (
	"testing"
	"github.com/bocheninc/L0/core/types"
	"time"
	"sync"
	"fmt"
)

func TestBlockNotify(t *testing.T) {
	var ttt sync.Map
	ttt.Store("111", time.Now())
	value, ok := ttt.Load("111")
	fmt.Println(value.(time.Time), ok)
}

func TestRegisterPublishTransaction(t *testing.T) {
	tx := &types.Transaction{}
	tx.Data.Nonce = 100

	time.Sleep(time.Second)
	RegisterTransaction(tx)
	time.Sleep(time.Second)
	PublishTransaction(tx)
	time.Sleep(5 * time.Second)
}


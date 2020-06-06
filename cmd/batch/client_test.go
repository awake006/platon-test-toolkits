package main

import (
	"context"
	"fmt"
	"math/big"
	"os"
	"os/signal"
	"syscall"
	"testing"

	"github.com/PlatONnetwork/PlatON-Go/common"
	"github.com/PlatONnetwork/PlatON-Go/ethclient"
)

var (
	accounts           = parseAccountFile(`all_addr_and_private_keys_bak.json`, 0, 2000, 10000)
	hasMoneyAddress    = "lax196278ns22j23awdfj9f2d4vz0pedld8au6xelj"
	hasMoneyPrivateKey = "a689f0879f53710e9e0c1025af410a530d6381eebb5916773195326e123b822b"
	urls               = []string{
		"ws://192.168.9.201:8808",
		"ws://192.168.9.201:8809",
		"ws://192.168.9.202:8808", "ws://192.168.9.202:8809",
		"ws://192.168.9.203:8808", "ws://192.168.9.203:8809",
		"ws://192.168.9.204:8808", "ws://192.168.9.204:8809",
	}
)

func TestClient(t *testing.T) {
	client, err := ethclient.Dial("ws://192.168.9.201:8809")
	if err != nil {
		panic(err.Error())
	}
	address, err := common.Bech32ToAddress(hasMoneyAddress)
	if err != nil {
		panic(err.Error())
	}
	nonce, err := client.NonceAt(context.Background(), address, nil)
	if err != nil {
		panic(err.Error())
	}
	number := big.NewInt(100)
	block, err := client.BlockByNumber(context.Background(), number)
	if err != nil {
		fmt.Println(err.Error())
	}
	fmt.Println(block.NumberU64())
	fmt.Println(nonce)
}

func TestBatchProcess_Start(t *testing.T) {
	bp := NewBatchProcess(accounts, urls, 5)
	bp.Start()
	sigs := make(chan os.Signal, 1)
	done := make(chan struct{}, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigs
		done <- struct{}{}
	}()
	<-done
	bp.Stop()
}

func TestNewBatchMixProcess_Start(t *testing.T) {
	nodekey := "d979cc4cb878676a134370d040766c74df1950e84996590cd0d6ea6b468af65d"
	bp := NewBatchMixProcess(accounts, urls, nodekey, true, 3)
	bp.Start()
	sigs := make(chan os.Signal, 1)
	done := make(chan struct{}, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigs
		done <- struct{}{}
	}()
	<-done
	bp.Stop()
}

func TestBatchTransfer_Start(t *testing.T) {
	url := urls[0]
	pk := "96bf130ca6c954e1cd49ff66f4f15b546da0a832f47678d7a22e2c7e928ce95f"
	bs := NewBatchTransfer(pk, url, accounts)
	bs.Start()
}

func TestBatchSameFromProcess_Start(t *testing.T) {
	bp := NewBatchSameFromProcess(accounts, urls, 3, 7)
	bp.Start()
	sigs := make(chan os.Signal, 1)
	done := make(chan struct{}, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigs
		done <- struct{}{}
	}()
	<-done
	bp.Stop()
}

func TestBatchProportionProcess_Start(t *testing.T) {
	bp := NewBatchProportionProcess(accounts, urls, 3, 7)
	bp.Start()
	sigs := make(chan os.Signal, 1)
	done := make(chan struct{}, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigs
		done <- struct{}{}
	}()
	<-done
	bp.Stop()
}

func TestBatchSameToProcess_Start(t *testing.T) {
	bp := NewBatchSameToProcess(accounts, urls, 3, 7)
	bp.Start()
	sigs := make(chan os.Signal, 1)
	done := make(chan struct{}, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigs
		done <- struct{}{}
	}()
	<-done
	bp.Stop()
}

func TestInit(t *testing.T) {
	fmt.Println(contractAddr.String())
}

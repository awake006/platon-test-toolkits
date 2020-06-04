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
	"github.com/PlatONnetwork/PlatON-Go/core/types"
	"github.com/PlatONnetwork/PlatON-Go/crypto"
	"github.com/PlatONnetwork/PlatON-Go/ethclient"
	"github.com/PlatONnetwork/PlatON-Go/params"
)

var (
	accounts           = parseAccountFile(`all_addr_and_private_keys.json`, 0, 2000, 10000)
	hasMoneyAddress    = "lax196278ns22j23awdfj9f2d4vz0pedld8au6xelj"
	hasMoneyPrivateKey = "a689f0879f53710e9e0c1025af410a530d6381eebb5916773195326e123b822b"
	urls               = []string{
		"ws://192.168.9.201:8808", "ws://192.168.9.201:8809",
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

func TestSendTransaction(t *testing.T) {
	client, err := ethclient.Dial("http://192.168.9.201:6789")
	if err != nil {
		panic(err.Error())
	}
	from, err := common.Bech32ToAddress(hasMoneyAddress)
	if err != nil {
		panic(err.Error())
	}
	signer := types.NewEIP155Signer(big.NewInt(ChainId))
	nonce, err := client.NonceAt(context.Background(), from, nil)
	if err != nil {
		panic(err.Error())
	}

	pk := crypto.HexMustToECDSA(hasMoneyPrivateKey)
	for _, account := range accounts {
		tx := types.NewTransaction(
			nonce,
			account.address,
			new(big.Int).Mul(big.NewInt(200), big.NewInt(params.LAT)),
			21000,
			big.NewInt(500000000000),
			nil)
		signedTx, err := types.SignTx(tx, signer, pk)
		if err != nil {
			fmt.Printf("sign tx error: %v\n", err)
			return
		}

		err = client.SendTransaction(context.Background(), signedTx)
		if err != nil {
			panic(err.Error())
		}
		nonce++
	}
}

func TestBatchParallelProcess_Start(t *testing.T) {
	bp := NewBatchParallelProcess(accounts, urls, 2, 7)
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

func TestBatchDifProcess_Start(t *testing.T) {
	bp := NewBatchDifProcess(accounts, urls, 3, 7)
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

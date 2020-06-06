package main

import (
	"context"
	"crypto/ecdsa"
	"fmt"
	"log"
	"math/big"
	"os"
	"time"

	"github.com/PlatONnetwork/PlatON-Go/core/types"
	"github.com/PlatONnetwork/PlatON-Go/crypto"
	"github.com/PlatONnetwork/PlatON-Go/ethclient"
	"github.com/PlatONnetwork/PlatON-Go/params"
)

type BatchTransfer struct {
	accounts AccountList
	url      string
	signer   types.EIP155Signer
	pk       *ecdsa.PrivateKey
}

func NewBatchTransfer(privateKey, url string, accounts AccountList) *BatchTransfer {
	return &BatchTransfer{
		accounts: accounts,
		url:      url,
		signer:   types.NewEIP155Signer(big.NewInt(ChainId)),
		pk:       crypto.HexMustToECDSA(privateKey),
	}
}

func (bs *BatchTransfer) Start() {
	var client *ethclient.Client
	var err error
	for {
		client, err = ethclient.Dial(bs.url)
		if err != nil {
			log.Printf("Failure to connect platon %s", err)
			time.Sleep(100 * time.Millisecond)
			continue
		}
		break
	}
	defer client.Close()

	for {
		number := big.NewInt(100)
		block, err := client.BlockByNumber(context.Background(), number)
		if err != nil {
			time.Sleep(100 * time.Millisecond)
		}
		if block != nil && err == nil {
			log.Println("Platon node mining now")
			break
		}
	}
	fmt.Println("start transfer")
	bs.transfer(client)
	fmt.Println("end transfer")
	os.Exit(0)
}

func (bs *BatchTransfer) transfer(client *ethclient.Client) {
	from := crypto.PubkeyToAddress(bs.pk.PublicKey)
	nonce, err := client.NonceAt(context.Background(), from, nil)
	if err != nil {
		panic(err.Error())
	}
	fmt.Println("nonce:", nonce)
	for i, account := range bs.accounts {
		tx := types.NewTransaction(
			nonce,
			account.address,
			new(big.Int).Mul(big.NewInt(20000), big.NewInt(params.LAT)),
			21000,
			big.NewInt(500000000000),
			nil)
		signedTx, err := types.SignTx(tx, bs.signer, bs.pk)
		if err != nil {
			fmt.Printf("sign tx error: %v,accounts index is:%d\n", err, i)
			panic(err.Error())
		}

		err = client.SendTransaction(context.Background(), signedTx)
		if err != nil {
			fmt.Printf("send tx error: %v,accounts index is:%d\n", err, i)
			panic(err.Error())
		}
		nonce++
		fmt.Println(tx.Hash().String())
	}
}

func (bs *BatchTransfer) Stop()                           {}
func (bs *BatchTransfer) Pause()                          {}
func (bs *BatchTransfer) Resume()                         {}
func (bs *BatchTransfer) SetSendInterval(d time.Duration) {}

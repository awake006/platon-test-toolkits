package main

import (
	"encoding/json"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/PlatONnetwork/PlatON-Go/common"
	"github.com/PlatONnetwork/PlatON-Go/crypto"
)

func TestAccount(t *testing.T) {
	var addrKeyList AddrKeyList
	for i := 0; i <= 1000000; i++ {
		privateKey, err := crypto.GenerateKey()
		if err != nil {
			t.Fatal(err.Error())
		}
		priByte := crypto.FromECDSA(privateKey)
		pri := common.Bytes2Hex(priByte)
		address := crypto.PubkeyToAddress(privateKey.PublicKey).String()
		addrKey := AddrKey{
			Address: address,
			Key:     pri,
		}

		addrKeyList = append(addrKeyList, addrKey)
	}
	file, err := os.Create("from_keys.json")
	if err != nil {
		t.Fatal(err.Error())
	}
	defer file.Close()
	buf, err := json.MarshalIndent(addrKeyList, " ", " ")
	if err != nil {
		t.Fatal(err.Error())
	}

	_, err = file.Write(buf)
	if err != nil {
		t.Error(err.Error())
	}
}

func TestNewBatchTransfer(t *testing.T) {
	fmt.Println("parse from address")
	accounts := parseAccountFile("./from.json", 0, 5000, 100000)
	fmt.Println("parse to address")
	toAccountList := parseToAccountFile("./all_addr_and_private_keys.json")
	urls := []string{
		"ws://10.1.1.25:6601", "ws://10.1.1.26:6601", "ws://10.1.1.27:6601", "ws://10.1.1.28:6601",
		"ws://10.1.1.29:6601", "ws://10.1.1.30:6601", "ws://10.1.1.31:6601", "ws://10.1.1.32:6601",
		"ws://10.1.1.33:6601", "ws://10.1.1.34:6601", "ws://10.1.1.35:6601", "ws://10.1.1.36:6601",
		"ws://10.1.1.37:6601", "ws://10.1.1.38:6601", "ws://10.1.1.39:6601", "ws://10.1.1.40:6601",
		"ws://10.1.1.41:6601", "ws://10.1.1.42:6601", "ws://10.1.1.43:6601", "ws://10.1.1.44:6601",
		"ws://10.1.1.45:6601", "ws://10.1.1.46:6601", "ws://10.1.1.47:6601", "ws://10.1.1.48:6601",
		"ws://10.1.1.49:6601",
	}
	batch := NewBatchTransfer(accounts, urls, 100, 100)
	fmt.Println("send address to toCh")

	done := make(chan struct{}, 1)
	over := false
	go func() {
		for _, acc := range toAccountList {
			batch.toAddrCh <- common.HexToAddress(acc.Address)
		}
		over = true
	}()
	fmt.Println("start")
	batch.Start()
	go func() {
		for {
			if over && len(batch.toAddrCh) == 0 {
				done <- struct{}{}
			}
			<-time.After(time.Second)
		}
	}()
	<-done
	batch.Stop()
}

func TestVersion(t *testing.T) {
	VersionMajor := 0
	VersionMinor := 12
	VersionPatch := 0
	GenesisVersion := uint32(VersionMajor<<16 | VersionMinor<<8 | VersionPatch)
	fmt.Println(GenesisVersion)
}

func TestTimer(t *testing.T) {
	fmt.Println("START")
	do := func() {
		fmt.Println("func")
	}
	time.AfterFunc(time.Second*3, do)
	time.Sleep(5 * time.Second)
}

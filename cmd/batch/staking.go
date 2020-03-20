package main

import (
	"context"
	"encoding/json"
	"io/ioutil"
	"log"
	"math/big"
	"time"

	"github.com/PlatONnetwork/PlatON-Go/core/types"
	"github.com/PlatONnetwork/PlatON-Go/crypto"
	"github.com/PlatONnetwork/PlatON-Go/ethclient"
	"github.com/awake006/platon-test-toolkits/util"
)

type BatchStaking struct {
	confList       StakingConfigList
	url            string
	programVersion uint32
	exit           chan struct{}
}
type StakingConfigList []*StakingConfig

type StakingConfig struct {
	Nodekey  string   `json:"nodekey"`
	Blskey   string   `json:"blskey"`
	NodeName string   `json:"node_name"`
	Addr     *AddrKey `json:"account"`
}

func NewBatchStaking(stakingFile, url string, programVersion uint32) *BatchStaking {
	stakingConfigList := parseStakingConfig(stakingFile)
	return &BatchStaking{
		confList:       stakingConfigList,
		url:            url,
		programVersion: programVersion,
	}
}

func (bs *BatchStaking) Start() {
	client, err := ethclient.Dial(bs.url)
	if err != nil {
		panic(err.Error())
	}
	for _, batchConf := range bs.confList {
		go bs.createStaking(client, batchConf)
	}
	timer := time.NewTimer(130 * time.Second)
	defer timer.Stop()
	for {
		select {
		case <-timer.C:
			return
		case <-bs.exit:
			return
		}
	}
}

func parseStakingConfig(stakingFile string) StakingConfigList {
	var stakingConfigList StakingConfigList
	b, err := ioutil.ReadFile(stakingFile)
	if err != nil {
		panic(err)
	}

	err = json.Unmarshal(b, &stakingConfigList)
	if err != nil {
		panic(err)
	}
	return stakingConfigList
}

func (bs *BatchStaking) createStaking(client *ethclient.Client, batchConf *StakingConfig) {
	stub := util.NewStakingStub(batchConf.Nodekey)
	buf, _ := stub.Create(batchConf.Blskey, batchConf.NodeName, bs.programVersion)
	signer := types.NewEIP155Signer(big.NewInt(ChainId))
	accountPK, err := crypto.HexToECDSA(batchConf.Addr.Key)
	if err != nil {
		log.Printf("%s parse account privatekey error: %v", stub.NodeID.String(), err)
		return
	}
	tx, err := types.SignTx(
		types.NewTransaction(
			0,
			contractAddr,
			big.NewInt(1),
			103496,
			big.NewInt(500000000000),
			buf),
		signer, accountPK)
	if err != nil {
		log.Printf("sign tx error: %v", err)
		return
	}

	err = client.SendTransaction(context.Background(), tx)
	if err != nil {
		log.Printf("%s send create staking transaction error %v", stub.NodeID.String(), err)
		return
	}

	t := time.Now()
	timer := time.NewTimer(100 * time.Millisecond)
	defer timer.Stop()
	for {
		select {
		case <-timer.C:
			_, err = client.TransactionReceipt(context.Background(), tx.Hash())
			if err == nil {
				log.Printf("%s create staking success!!!\n", stub.NodeID.String())
				return
			}
			if time.Since(t) > 120*time.Second {
				log.Printf("%s Get transaction receipt timeout %s", stub.NodeID.String(), tx.Hash().String())
				return
			}
			timer.Reset(100 * time.Millisecond)
		case <-bs.exit:
			return
		}
	}
}

func (bs *BatchStaking) Stop() {
	close(bs.exit)
}
func (bs *BatchStaking) Pause() {

}
func (bs *BatchStaking) Resume() {}

func (bs *BatchStaking) SetSendInterval(d time.Duration) {}

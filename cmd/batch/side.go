package main

import (
	"context"
	"log"
	"math/big"
	"time"

	"github.com/PlatONnetwork/PlatON-Go/core/types"
	"github.com/PlatONnetwork/PlatON-Go/ethclient"
	"github.com/PlatONnetwork/PlatON-Go/rpc"
	"github.com/awake006/platon-test-toolkits/util"
)

var (
	stopNumber            = 40000
	consensusSendInterval = 50 * time.Millisecond
	defaultSendInterval   = 1 * time.Second
)

type SideBatch struct {
	process BatchProcessor

	url string

	exit chan struct{}

	account  *Account
	nodeKey  string
	blsKey   string
	nodeName string

	onlyConsensus bool
	staking       bool
}

func NewSideBatch(
	process BatchProcessor,
	account *Account,
	url, nodeKey, blsKey, nodeName string,
	onlyConsensus, staking bool) *SideBatch {
	return &SideBatch{
		process:       process,
		url:           url,
		exit:          make(chan struct{}),
		account:       account,
		nodeKey:       nodeKey,
		blsKey:        blsKey,
		nodeName:      nodeName,
		onlyConsensus: onlyConsensus,
		staking:       staking,
	}
}

func (sb *SideBatch) Start() {
	sb.checkMining()
	sb.process.Start()

	if sb.onlyConsensus {
		go sb.loop()
	}
}

func (sb *SideBatch) Stop() {
	close(sb.exit)
	sb.process.Stop()
}

func (sb *SideBatch) Pause()  {}
func (sb *SideBatch) Resume() {}

func (sb *SideBatch) SetSendInterval(d time.Duration) {
}

func (sb *SideBatch) checkMining() {
	var client *ethclient.Client
	var err error
	for {
		client, err = ethclient.Dial(sb.url)
		if err != nil {
			log.Printf("Failure to connect platon %s", err)
			time.Sleep(100 * time.Millisecond)
			continue
		}
		break
	}
	defer client.Close()

	staking := false
	stub := util.NewStakingStub(sb.nodeKey)

	for {
		if !staking && sb.staking {
			if _, err := client.BlockByNumber(context.Background(), big.NewInt(5)); err != nil {
				time.Sleep(100 * time.Millisecond)
				continue
			}

			sb.createStaking(client, stub, sb.account)
			staking = true
		}

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
}

func (sb *SideBatch) loop() {
	// Check if mining block has arrived the stop number.
	//
	var client *rpc.Client
	var err error
	for {
		client, err = rpc.Dial(sb.url)
		if err != nil {
			log.Printf("Failure to connect platon %s", err)
			time.Sleep(100 * time.Millisecond)
			continue
		}
		break
	}
	defer client.Close()

	timer := time.NewTicker(1 * time.Second)

	/*
		stopNumber := big.NewInt(int64(stopNumber))
		arrived := func() bool {
			block, err := client.BlockByNumber(context.Background(), stopNumber)
			if err != nil {
				return false
			}
			if block != nil && err == nil {
				return true
			}
			return false
		}*/

	consensus := func() bool {
		var c bool
		err := client.Call(&c, "debug_isConsensusNode")
		if err != nil {
			log.Printf("Failure call debug_isConsensusNode%s", err)
			return false
		}
		// log.Printf("Current node is proposer? %v", proposer)
		return c

	}

	for {
		select {
		case <-sb.exit:
			return
		case <-timer.C:
			if consensus() {
				// sb.process.SetSendInterval(consensusSendInterval)
				sb.process.Resume()
			} else {
				// sb.process.SetSendInterval(defaultSendInterval)
				sb.process.Pause()
			}
		}
	}
}

func (sb *SideBatch) createStaking(client *ethclient.Client, stub *util.StakingStub, account *Account) {
	buf, _ := stub.Create(sb.blsKey, sb.nodeName)
	signer := types.NewEIP155Signer(big.NewInt(ChainId))
	tx, err := types.SignTx(
		types.NewTransaction(
			0,
			contractAddr,
			big.NewInt(1),
			103496,
			big.NewInt(500000000000),
			buf),
		signer,
		account.privateKey)
	if err != nil {
		log.Printf("sign tx error: %v", err)
		return
	}

	err = client.SendTransaction(context.Background(), tx)
	if err != nil {
		log.Printf("send create staking transaction error %v", err)
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
				log.Println("create staking success!!!")
				return
			}
			if time.Since(t) > 120*time.Second {
				log.Printf("Get transaction receipt timeout %s", tx.Hash().String())
				return
			}
			timer.Reset(100 * time.Millisecond)
		case <-sb.exit:
			return
		}
	}
}

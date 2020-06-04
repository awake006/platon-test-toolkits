package main

import (
	"context"
	"fmt"
	"math/big"
	"sync"
	"sync/atomic"
	"time"

	"github.com/PlatONnetwork/PlatON-Go/common"
	"github.com/PlatONnetwork/PlatON-Go/core/types"
	"github.com/PlatONnetwork/PlatON-Go/crypto"
	"github.com/PlatONnetwork/PlatON-Go/ethclient"
)

type BatchProportionProcess struct {
	accounts AccountList
	hosts    []string

	sendCh   chan *Account
	waitCh   chan *ReceiptTask
	acceptCh chan common.Address
	signer   types.EIP155Signer

	exit chan struct{}

	sents    int32
	receipts int32

	sendInterval atomic.Value // time.Duration

	paused      bool
	lock        sync.Mutex
	cond        *sync.Cond
	maxSendTxns int
	proportion  int

	BatchProcessor
}

func NewBatchProportionProcess(accounts AccountList, hosts []string, maxSendTxns, proportion int) *BatchProportionProcess {
	bp := &BatchProportionProcess{
		accounts:    accounts,
		hosts:       hosts,
		sendCh:      make(chan *Account, len(accounts)*2),
		acceptCh:    make(chan common.Address, len(accounts)*2),
		waitCh:      make(chan *ReceiptTask, len(accounts)*2),
		signer:      types.NewEIP155Signer(big.NewInt(ChainId)),
		exit:        make(chan struct{}),
		sents:       0,
		paused:      false,
		maxSendTxns: maxSendTxns,
		proportion:  proportion,
	}
	bp.cond = sync.NewCond(&bp.lock)
	bp.sendInterval.Store(50 * time.Millisecond)
	return bp
}

func (bp *BatchProportionProcess) Start() {
	fmt.Println("Start generate accept account")
	bp.GenAcceptAccount()
	fmt.Println("Generate accept account ok")
	go bp.report()

	for _, host := range bp.hosts {
		go bp.perform(host)
	}

	for _, act := range bp.accounts {
		bp.sendCh <- act
		time.Sleep(100 * time.Millisecond)
	}
	fmt.Println("start success")
}

func (bp *BatchProportionProcess) Stop() {
	close(bp.exit)
}

func (bp *BatchProportionProcess) Pause() {
	bp.cond.L.Lock()
	defer bp.cond.L.Unlock()
	bp.paused = true
}

func (bp *BatchProportionProcess) Resume() {
	bp.cond.L.Lock()
	defer bp.cond.L.Unlock()
	if !bp.paused {
		return
	}
	bp.paused = false
	bp.cond.Signal()
}

func (bp *BatchProportionProcess) GenAcceptAccount() {
	for i := 0; i < len(bp.accounts); i++ {
		pk, err := crypto.GenerateKey()
		if err != nil {
			panic(err.Error())
		}
		address := crypto.PubkeyToAddress(pk.PublicKey)
		bp.acceptCh <- address
	}
}

func (bp *BatchProportionProcess) SetSendInterval(d time.Duration) {
	bp.sendInterval.Store(d)
}

func (bp *BatchProportionProcess) report() {
	timer := time.NewTimer(time.Second)
	for {
		select {
		case <-timer.C:
			cnt := atomic.SwapInt32(&bp.sents, 0)
			receipts := atomic.SwapInt32(&bp.receipts, 0)
			fmt.Printf("Send: %d/s, Receipts: %d/s\n", cnt, receipts)
			timer.Reset(time.Second)
		case <-bp.exit:
			return
		}
	}
}

func (bp *BatchProportionProcess) perform(host string) {
	client, err := ethclient.Dial(host)
	if err != nil {
		panic(err)
	}
	defer client.Close()
	count := 0
	for {
		bp.cond.L.Lock()
		if bp.paused {
			bp.cond.Wait()
		}
		bp.cond.L.Unlock()

		select {
		case act := <-bp.sendCh:
			if count < bp.proportion {
				bp.sendTransaction(client, act, 1)
				count++
			} else {
				count = 0
				bp.sendTransaction(client, act, bp.maxSendTxns)
			}

		case task := <-bp.waitCh:
			bp.getTransactionReceipt(client, task)
		case <-bp.exit:
			return
		}
	}
}

func (bp *BatchProportionProcess) nonceAt(client *ethclient.Client, addr common.Address) uint64 {
	var blockNumber *big.Int
	nonce, err := client.NonceAt(context.Background(), addr, blockNumber)
	if err != nil {
		fmt.Printf("Get nonce error, addr: %s, err:%v\n", addr, err)
		return 0
	}
	return nonce
}

func (bp *BatchProportionProcess) sendTransaction(client *ethclient.Client, account *Account, sendTxs int) {
	to := <-bp.acceptCh
	nonce := bp.nonceAt(client, account.address)
	for i := 0; i < sendTxs; i++ {
		tx := types.NewTransaction(
			nonce,
			to,
			big.NewInt(200),
			21000,
			big.NewInt(500000000000),
			nil)
		signedTx, err := types.SignTx(tx, bp.signer, account.privateKey)
		if err != nil {
			fmt.Printf("sign tx error: %v\n", err)
			bp.sendCh <- account
			bp.acceptCh <- to
			return
		}

		err = client.SendTransaction(context.Background(), signedTx)
		account.lastSent = time.Now()
		if err != nil {
			fmt.Printf("send transaction error: %v\n", err)
			go func() {
				<-time.After(bp.sendInterval.Load().(time.Duration))
				bp.sendCh <- account
				bp.acceptCh <- to
			}()
			return
		}
		// account.nonce = nonce + 1
		atomic.AddInt32(&bp.sents, 1)

		nonce += 1

		if i < sendTxs-1 {
			continue
		}

		go func() {
			<-time.After(2 * time.Second)
			bp.waitCh <- &ReceiptTask{
				account: account,
				hash:    signedTx.Hash(),
				to:      to,
			}
		}()
	}
}

func (bp *BatchProportionProcess) getTransactionReceipt(client *ethclient.Client, task *ReceiptTask) {
	// fmt.Println("get receipts:", task.to.String())
	_, err := client.TransactionReceipt(context.Background(), task.hash)
	if err != nil {
		if time.Since(task.account.lastSent) >= task.account.interval {
			fmt.Printf("get receipt timeout, address:%s, hash: %s, sendTime: %v, now: %v\n",
				task.account.address.String(), task.hash.String(), task.account.lastSent, time.Now())
			bp.sendCh <- task.account
			bp.acceptCh <- task.to
			return
		}
		go func() {
			<-time.After(300 * time.Millisecond)
			bp.waitCh <- task
		}()
		return
	}

	atomic.AddInt32(&bp.receipts, 1)

	go func() {
		<-time.After(bp.sendInterval.Load().(time.Duration))
		bp.sendCh <- task.account
		bp.acceptCh <- task.to
	}()
}

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

type BatchSameFromProcess struct {
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

func NewBatchSameFromProcess(accounts AccountList, hosts []string, maxSendTxns, proportion int) *BatchSameFromProcess {
	bp := &BatchSameFromProcess{
		accounts:    accounts,
		hosts:       hosts,
		sendCh:      make(chan *Account, len(accounts)),
		acceptCh:    make(chan common.Address, len(accounts)*maxSendTxns),
		waitCh:      make(chan *ReceiptTask, len(accounts)*maxSendTxns),
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

func (bp *BatchSameFromProcess) Start() {
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

func (bp *BatchSameFromProcess) Stop() {
	close(bp.exit)
}

func (bp *BatchSameFromProcess) Pause() {
	bp.cond.L.Lock()
	defer bp.cond.L.Unlock()
	bp.paused = true
}

func (bp *BatchSameFromProcess) Resume() {
	bp.cond.L.Lock()
	defer bp.cond.L.Unlock()
	if !bp.paused {
		return
	}
	bp.paused = false
	bp.cond.Signal()
}

func (bp *BatchSameFromProcess) GenAcceptAccount() {
	for i := 0; i < cap(bp.acceptCh); i++ {
		pk, err := crypto.GenerateKey()
		if err != nil {
			panic(err.Error())
		}
		address := crypto.PubkeyToAddress(pk.PublicKey)
		bp.acceptCh <- address
	}
}

func (bp *BatchSameFromProcess) SetSendInterval(d time.Duration) {
	bp.sendInterval.Store(d)
}

func (bp *BatchSameFromProcess) report() {
	timer := time.NewTimer(time.Second)
	for {
		select {
		case <-timer.C:
			cnt := atomic.SwapInt32(&bp.sents, 0)
			receipts := atomic.SwapInt32(&bp.receipts, 0)
			fmt.Printf("sendCh:%d, acceptCh:%d, waitCh:%d\n", len(bp.sendCh), len(bp.acceptCh), len(bp.waitCh))
			fmt.Printf("Send: %d/s, Receipts: %d/s\n", cnt, receipts)
			timer.Reset(time.Second)
		case <-bp.exit:
			return
		}
	}
}

func (bp *BatchSameFromProcess) perform(host string) {
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
			nonce := bp.nonceAt(client, act.address)
			if count < bp.proportion {
				if err := bp.sendDifFromAddrTransaction(client, act, nonce); err == nil {
					count++
				}
			} else {
				if err := bp.sendSameFromAddrTransaction(client, act, nonce); err == nil {
					count = 0
				}
			}
		case task := <-bp.waitCh:
			bp.getTransactionReceipt(client, task)
		case <-bp.exit:
			return
		}
	}
}

func (bp *BatchSameFromProcess) sendSameFromAddrTransaction(client *ethclient.Client, act *Account, nonce uint64) error {
	for i := 0; i < bp.maxSendTxns; i++ {
		to := <-bp.acceptCh
		hash, err := bp.sendTransaction(client, act, to, nonce)
		if err != nil {
			go func() {
				<-time.After(500 * time.Millisecond)
				bp.acceptCh <- to
				bp.sendCh <- act
			}()
			return err
		}
		go func() {
			<-time.After(2 * time.Second)
			bp.acceptCh <- to
		}()
		nonce++
		if i < bp.maxSendTxns-1 {
			continue
		}
		go func() {
			<-time.After(2 * time.Second)
			bp.waitCh <- &ReceiptTask{
				account: act,
				hash:    hash,
			}
		}()
	}
	return nil

}

func (bp *BatchSameFromProcess) sendDifFromAddrTransaction(client *ethclient.Client, act *Account, nonce uint64) error {
	to := <-bp.acceptCh
	hash, err := bp.sendTransaction(client, act, to, nonce)

	if err != nil {
		go func() {
			<-time.After(500 * time.Millisecond)
			bp.sendCh <- act
			bp.acceptCh <- to
		}()
		return err
	}
	go func() {
		<-time.After(2 * time.Second)
		bp.acceptCh <- to
		bp.waitCh <- &ReceiptTask{
			account: act,
			hash:    hash,
		}
	}()
	return nil
}

func (bp *BatchSameFromProcess) nonceAt(client *ethclient.Client, addr common.Address) uint64 {
	var blockNumber *big.Int
	nonce, err := client.NonceAt(context.Background(), addr, blockNumber)
	if err != nil {
		fmt.Printf("Get nonce error, addr: %s, err:%v\n", addr, err)
		return 0
	}
	return nonce
}

func (bp *BatchSameFromProcess) sendTransaction(client *ethclient.Client, account *Account, to common.Address, nonce uint64) (common.Hash, error) {
	tx := types.NewTransaction(
		nonce,
		to,
		big.NewInt(200),
		21000,
		big.NewInt(500000000000),
		nil)

	signedTx, err := types.SignTx(tx, bp.signer, account.privateKey)
	if err != nil {
		return common.Hash{}, err
	}
	err = client.SendTransaction(context.Background(), signedTx)
	account.lastSent = time.Now()
	if err != nil {
		fmt.Printf("send transaction error: %v\n", err)
		return common.Hash{}, err
	}
	atomic.AddInt32(&bp.sents, 1)
	// fmt.Printf("from:%s,to:%s\n",account.address.String(), to.String())
	return signedTx.Hash(), nil
}

func (bp *BatchSameFromProcess) getTransactionReceipt(client *ethclient.Client, task *ReceiptTask) {
	// fmt.Println("get receipts:", task.to.String())
	_, err := client.TransactionReceipt(context.Background(), task.hash)
	if err != nil {
		if time.Since(task.account.lastSent) >= task.account.interval {
			fmt.Printf("get receipt timeout, address:%s, hash: %s, sendTime: %v, now: %v\n",
				task.account.address.String(), task.hash.String(), task.account.lastSent, time.Now())
			bp.sendCh <- task.account
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
	}()
}

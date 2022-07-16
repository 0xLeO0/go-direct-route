package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math/big"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/params"

	"github.com/node-real/go-direct-route/client"
	eabi "github.com/node-real/go-direct-route/example/abi"
	"github.com/node-real/go-direct-route/example/utils"
)

var rpcEndPoint = "https://bsc-dataseed.binance.org"

// Testnet endpoint
//var rpcEndPoint = "https://data-seed-prebsc-1-s1.binance.org:8545"

var directRouteEndPoint = "https://api.nodereal.io/direct-route"

// Testnet endpoint
//var directRouteEndPoint = "https://testnet-api.nodereal.io/direct-route"

// var account1, _ = utils.FromHexKey("299d9a2ceda42af9ae8936fdafc680a764f324689a3d0f516ef3cad5048b692b")
// var account2, _ = utils.FromHexKey("ac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80")

var account1, _ = utils.FromHexKey("6e22bd008464f3c6a045b711fdc696d5afab7e95594ef299013a930a1a916f68")
var account2, _ = utils.FromHexKey("299d9a2ceda42af9ae8936fdafc680a764f324689a3d0f516ef3cad5048b692b")

func getBundlePriceDemo() {
	// Initialize the direct route client
	client, _ := client.Dial(directRouteEndPoint)
	price, err := client.BundlePrice(context.Background())
	if err != nil {
		log.Fatal(fmt.Sprintf("failed to get bundle price %v", err))
	}
	fmt.Printf("get bundle price price %s\n", price.String())
}

func getServiceStatus() {
	directClient, _ := client.Dial(directRouteEndPoint)
	s, err := directClient.GetStatus(context.Background())
	if err != nil {
		panic(err)
	}
	bz, _ := json.Marshal(s)
	fmt.Println(string(bz))
}

/**
In this case, we try to use two accounts to send BNB to each other,
the two transaction should be all success or all failed.
*/
func sendBNBByBundleDemo() {
	directClient, _ := client.Dial(directRouteEndPoint)
	rpcClient, _ := ethclient.Dial(rpcEndPoint)
	price, err := directClient.BundlePrice(context.Background())
	if err != nil {
		log.Fatalf("failed to get bundle price%v", err)
	}

	n1, _ := rpcClient.PendingNonceAt(context.Background(), account1.Addr)

	chainId := big.NewInt(56)

	// testnet chain id
	//chainId := big.NewInt(97)
	valueToTransfer := big.NewInt(100 * params.GWei)
	gasLimit := uint64(23000)

	tx1, hash1, _ := utils.SignTransaction(account1, account2.Addr, valueToTransfer, nil, n1, gasLimit, price, chainId)
	tx2, hash2, _ := utils.SignTransaction(account1, account2.Addr, valueToTransfer, nil, n1+1, gasLimit, price, chainId)

	maxTime := uint64(time.Now().Unix() + 80)

	bundle := &client.SendBundleArgs{
		Txs:               []string{hexutil.Encode(tx1), hexutil.Encode(tx2)},
		MaxBlockNumber:    "",
		MinTimestamp:      nil,
		MaxTimestamp:      &maxTime,
		RevertingTxHashes: nil,
	}
	bundleHash, err := directClient.SendBundle(context.Background(), bundle)
	if err != nil {
		log.Fatalf("failed to send bundle %v", err)
	}
	fmt.Printf("successfull send bundle, hash %v\n", bundleHash)

	queryBundle, err := directClient.GetBundleByHash(context.Background(), bundleHash)
	if err != nil || queryBundle == nil {
		log.Fatalf("failed to query bundle %v", err)
	}

	bz, _ := json.Marshal(queryBundle)
	fmt.Printf("The bundle is %s\n", string(bz))

	found := false
	for i := 0; i < 42; i++ {
		r1, err1 := rpcClient.TransactionReceipt(context.Background(), hash1)
		r2, err2 := rpcClient.TransactionReceipt(context.Background(), hash2)
		if r1 != nil && err1 == nil && r2 != nil && err2 == nil {
			found = true
			break
		}
		time.Sleep(3 * time.Second)
	}
	if found {
		fmt.Println("bundle verified on chain")
	} else {
		log.Fatalf("bundle failed to be verified on chain or timeout")
	}
}

/**
In this case, we try to use two accounts to send BUSD to each other,
the second transaction are allowed to be failed,
we want the bundle been verified on chain during [now+20 second, now+80 second].
*/
func sendBUSDByBundleDemo() {
	directClient, _ := client.Dial(directRouteEndPoint)
	rpcClient, _ := ethclient.Dial(rpcEndPoint)
	price, _ := directClient.BundlePrice(context.Background())

	n1, _ := rpcClient.PendingNonceAt(context.Background(), account1.Addr)

	chainId := big.NewInt(56)

	// testnet chain id
	//chainId := big.NewInt(97)
	valueToTransfer := big.NewInt(0)
	gasLimit := uint64(70000)

	bep20ABI, _ := abi.JSON(strings.NewReader(eabi.BEP20ABI))

	data1, _ := bep20ABI.Pack("transfer", account2.Addr, big.NewInt(1))
	data2, _ := bep20ABI.Pack("transfer", account2.Addr, big.NewInt(1))

	tx1, hash1, _ := utils.SignTransaction(account1, common.HexToAddress("0xe9e7cea3dedca5984780bafc599bd69add087d56"), valueToTransfer, data1, n1, gasLimit, price, chainId)
	tx2, hash2, _ := utils.SignTransaction(account1, common.HexToAddress("0xe9e7cea3dedca5984780bafc599bd69add087d56"), valueToTransfer, data2, n1+1, gasLimit, price, chainId)

	maxTime := uint64(time.Now().Unix() + 80)
	minTime := uint64(time.Now().Unix() + 20)

	bundle := &client.SendBundleArgs{
		Txs:               []string{hexutil.Encode(tx1), hexutil.Encode(tx2)},
		MaxBlockNumber:    "",
		MinTimestamp:      &minTime,
		MaxTimestamp:      &maxTime,
		RevertingTxHashes: []common.Hash{hash2},
	}
	bundleHash, err := directClient.SendBundle(context.Background(), bundle)
	if err != nil {
		log.Fatalf("failed to send bundle %v", err)
	}
	fmt.Printf("successfull send bundle, hash %v\n", bundleHash)

	queryBundle, err := directClient.GetBundleByHash(context.Background(), bundleHash)
	if err != nil || queryBundle == nil {
		log.Fatalf("failed to query bundle %v", err)
	}

	bz, _ := json.Marshal(queryBundle)
	fmt.Printf("The bundle is %s\n", string(bz))

	found := false
	for i := 0; i < 60; i++ {
		r1, err1 := rpcClient.TransactionReceipt(context.Background(), hash1)
		r2, err2 := rpcClient.TransactionReceipt(context.Background(), hash2)
		if r1 != nil && err1 == nil && r2 != nil && err2 == nil {
			found = true
			break
		}
		time.Sleep(3 * time.Second)
	}
	if found {
		fmt.Println("bundle verified on chain")
	} else {
		log.Fatalf("bundle failed to be verified on chain or timeout")
	}
}

func sendBNBAndTokenBundleDemo() {
	directClient, _ := client.Dial(directRouteEndPoint)
	rpcClient, _ := ethclient.Dial(rpcEndPoint)
	price, err := directClient.BundlePrice(context.Background())
	if err != nil {
		log.Fatalf("failed to get bundle price%v", err)
	}

	n2, _ := rpcClient.PendingNonceAt(context.Background(), account2.Addr)

	chainId := big.NewInt(56)

	// testnet chain id
	//chainId := big.NewInt(97)
	valueToTransfer := big.NewInt(0 * params.GWei)
	gasLimit := uint64(500000)

	bep20ABI, _ := abi.JSON(strings.NewReader(eabi.BEP20ABI))
	// data2, _ := bep20ABI.Pack("transfer", account1.Addr, big.NewInt(7350000000000000))
	data2, _ := bep20ABI.Pack("transfer", account1.Addr, big.NewInt(1))

	for i := 0; i < 1000; i++ {
		tx2, _, _ := utils.SignTransaction(account2, common.HexToAddress("0xe9e7cea3dedca5984780bafc599bd69add087d56"), valueToTransfer, data2, n2 + uint64(i), gasLimit, price, chainId)

		maxTime := uint64(time.Now().Unix() + 80)
	
		bundle := &client.SendBundleArgs{
			Txs:               []string{ hexutil.Encode(tx2)},
			MaxBlockNumber:    "",
			MinTimestamp:      nil,
			MaxTimestamp:      &maxTime,
			RevertingTxHashes: nil,
		}
		bundleHash, err := directClient.SendBundle(context.Background(), bundle)
		if err != nil {
			log.Fatalf("failed to send bundle %v", err)
		}
		fmt.Printf("successfull send bundle, hash %v\n", bundleHash)

		queryBundle, err := directClient.GetBundleByHash(context.Background(), bundleHash)
		if err != nil || queryBundle == nil {
			log.Fatalf("failed to query bundle %v", err)
		}

		bz, _ := json.Marshal(queryBundle)
		fmt.Printf("The bundle is %s\n", string(bz))

	}
	// found := false
	// for i := 0; i < 42; i++ {
	// 	r1, err1 := rpcClient.TransactionReceipt(context.Background(), hash1)
	// 	r2, err2 := rpcClient.TransactionReceipt(context.Background(), hash2)
	// 	if r1 != nil && err1 == nil && r2 != nil && err2 == nil {
	// 		found = true
	// 		break
	// 	}
	// 	time.Sleep(3 * time.Second)
	// }
	// if found {
	// 	fmt.Println("bundle verified on chain")
	// } else {
	// 	log.Fatalf("bundle failed to be verified on chain or timeout")
	// }
}

func main() {
	sendBNBAndTokenBundleDemo()
}

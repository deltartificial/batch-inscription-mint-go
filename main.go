package main

import (
	"context"
	"crypto/ecdsa"
	"fmt"
	"log"
	"math/big"
	"os"
	"strconv"
	"sync"
	"time"

	"github.com/joho/godotenv"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
)

/// @title 📜 Batch Inscription Mint in Golang
/// @dev Send multiple & automatized mint transactions in Go, for all EVM networks.
/// @author deltartificial

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}

	httpRPC := os.Getenv("HTTP_RPC_URL")
	privateKeyHex := os.Getenv("PRIVATE_KEY_HEX")
	numWorkersStr := os.Getenv("NUM_WORKERS")
	transactionsNumberStr := os.Getenv("TRANSACTIONS_NUMBER")
	jsonData := os.Getenv("JSON_DATA")

	fmt.Println("--------------------------------")
	fmt.Println("@author : deltartificial")
	fmt.Println("Data to send :", jsonData)
	fmt.Println("Number of transactions to send :", transactionsNumberStr)
	fmt.Println("--------------------------------")

	numWorkers, err := strconv.Atoi(numWorkersStr)
	if err != nil {
		log.Fatal("error converting NUM_WORKERS to int:", err)
	}

	transactionsNumber, err := strconv.Atoi(transactionsNumberStr)
	if err != nil {
		log.Fatal("error converting TRANSACTIONS_NUMBER to int:", err)
	}

	client, err := ethclient.Dial(httpRPC)
	if err != nil {
		log.Fatal("error connecting to eth client:", err)
	}

	privateKey, err := crypto.HexToECDSA(privateKeyHex)
	if err != nil {
		log.Fatal("error reading private key:", err)
	}

	publicKey := privateKey.Public()
	publicKeyECDSA, ok := publicKey.(*ecdsa.PublicKey)
	if !ok {
		log.Fatal("error casting public key to ECDSA")
	}

	fromAddress := crypto.PubkeyToAddress(*publicKeyECDSA)
	nonce, err := client.PendingNonceAt(context.Background(), fromAddress)
	if err != nil {
		log.Fatal("error getting nonce:", err)
	}

	txnsChan := make(chan int, transactionsNumber)
	nonceChan := make(chan uint64, transactionsNumber)
	for i := nonce; i < nonce+uint64(transactionsNumber); i++ {
		nonceChan <- i
	}

	var wg sync.WaitGroup

	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go worker(client, privateKey, fromAddress, &wg, txnsChan, nonceChan, jsonData)
	}

	for i := 0; i < transactionsNumber; i++ {
		txnsChan <- i
		time.Sleep(1 * time.Second)
	}
	close(txnsChan)
	close(nonceChan)

	wg.Wait()
	fmt.Println("✨ All transactions sent.")
}

func worker(client *ethclient.Client, privateKey *ecdsa.PrivateKey, fromAddress common.Address, wg *sync.WaitGroup, txnsChan <-chan int, nonceChan <-chan uint64, jsonData string) {
	defer wg.Done()

	for _ = range txnsChan {
		nonce := <-nonceChan
		sendTransaction(client, privateKey, fromAddress, jsonData, nonce)
	}
}

func sendTransaction(client *ethclient.Client, privateKey *ecdsa.PrivateKey, fromAddress common.Address, jsonData string, nonce uint64) {
	gasPrice, err := client.SuggestGasPrice(context.Background())
	if err != nil {
		log.Fatal("error getting gas price:", err)
		return
	}

	value := big.NewInt(0) 
	gasLimit := uint64(22000)
	data := []byte(jsonData)

	tx := types.NewTransaction(nonce, fromAddress, value, gasLimit, gasPrice, data)

	

	chainID, err := client.NetworkID(context.Background())
	if err != nil {
		log.Fatal("error getting network ID:", err)
		return
	}

	signedTx, err := types.SignTx(tx, types.NewEIP155Signer(chainID), privateKey)
	if err != nil {
		log.Fatal("error signing transaction:", err)
		return
	}

	err = client.SendTransaction(context.Background(), signedTx)
	if err != nil {
		log.Fatal("error sending transaction:", err)
		return
	}

	fmt.Printf("📜 Transaction sent: %s\n", signedTx.Hash().Hex())
}
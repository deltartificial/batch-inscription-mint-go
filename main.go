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

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/joho/godotenv"
)

func main() {
	if err := godotenv.Load(); err != nil {
		log.Fatal("Error loading .env file")
	}

	httpRPC, privateKeyHex, numWorkersStr, transactionsNumberStr, jsonData := getConfig()

	client, privateKey, fromAddress := ethereumSetup(httpRPC, privateKeyHex)

	nonce := getInitialNonce(client, fromAddress)

	txnsChan := make(chan int, transactionsNumberStr)
	defer close(txnsChan)

	gasPrice := getGasPrice(client)

	var wg sync.WaitGroup
	for i := 0; i < numWorkersStr; i++ {
		wg.Add(1)
		go worker(client, privateKey, fromAddress, &wg, txnsChan, jsonData, nonce, gasPrice)
		nonce += uint64(transactionsNumberStr) / uint64(numWorkersStr)
	}

	for i := 0; i < transactionsNumberStr; i++ {
		txnsChan <- i
		time.Sleep(1 * time.Second)
	}

	wg.Wait()
	fmt.Println("✨ All transactions sent.")
}

func getConfig() (httpRPC, privateKeyHex string, numWorkersStr, transactionsNumberStr int, jsonData string) {
	httpRPC = os.Getenv("HTTP_RPC_URL")
	privateKeyHex = os.Getenv("PRIVATE_KEY_HEX")
	numWorkersStr, _ = strconv.Atoi(os.Getenv("NUM_WORKERS"))
	transactionsNumberStr, _ = strconv.Atoi(os.Getenv("TRANSACTIONS_NUMBER"))
	jsonData = os.Getenv("JSON_DATA")
	return
}

func ethereumSetup(httpRPC, privateKeyHex string) (*ethclient.Client, *ecdsa.PrivateKey, common.Address) {
	client, err := ethclient.Dial(httpRPC)
	if err != nil {
		log.Fatal("error connecting to eth client:", err)
	}

	privateKey, err := crypto.HexToECDSA(privateKeyHex)
	if err != nil {
		log.Fatal("error reading private key:", err)
	}

	publicKeyECDSA, ok := privateKey.Public().(*ecdsa.PublicKey)
	if !ok {
		log.Fatal("error casting public key to ECDSA")
	}

	fromAddress := crypto.PubkeyToAddress(*publicKeyECDSA)
	return client, privateKey, fromAddress
}

func getInitialNonce(client *ethclient.Client, fromAddress common.Address) uint64 {
	nonce, err := client.PendingNonceAt(context.Background(), fromAddress)
	if err != nil {
		log.Fatal("error getting nonce:", err)
	}
	return nonce
}

func getGasPrice(client *ethclient.Client) *big.Int {
	gasPrice, err := client.SuggestGasPrice(context.Background())
	if err != nil {
		log.Fatal("error getting gas price:", err)
	}
	return gasPrice
}

func worker(client *ethclient.Client, privateKey *ecdsa.PrivateKey, fromAddress common.Address, wg *sync.WaitGroup, txnsChan <-chan int, jsonData string, startNonce uint64, gasPrice *big.Int) {
	defer wg.Done()

	nonce := startNonce
	for range txnsChan {
		sendTransaction(client, privateKey, fromAddress, jsonData, nonce, gasPrice)
		nonce++ 
	}
}

func sendTransaction(client *ethclient.Client, privateKey *ecdsa.PrivateKey, fromAddress common.Address, jsonData string, nonce uint64, gasPrice *big.Int) {
	value := big.NewInt(0)
	gasLimit := uint64(22000)
	data := []byte(jsonData)

	tx := types.NewTransaction(nonce, fromAddress, value, gasLimit, gasPrice, data)

	chainID, err := client.NetworkID(context.Background())
	if err != nil {
		log.Printf("error getting network ID: %v", err)
		return
	}

	signedTx, err := types.SignTx(tx, types.NewEIP155Signer(chainID), privateKey)
	if err != nil {
		log.Printf("error signing transaction: %v", err)
		return
	}

	if err := client.SendTransaction(context.Background(), signedTx); err != nil {
		log.Printf("error sending transaction: %v", err)
		return
	}

	fmt.Printf("📜 Transaction sent: %s\n", signedTx.Hash().Hex())
}

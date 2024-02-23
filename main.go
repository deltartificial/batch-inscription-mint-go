package main

import (
	"context"
	"crypto/ecdsa"
	"fmt"
	"log"
	"math/big"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/joho/godotenv"
)

var rpcURLs = []string{
	"https://ava-mainnet.public.blastapi.io/ext/bc/C/rpc",
	"https://avalanche.blockpi.network/v1/rpc/public",
	"https://avax.meowrpc.com",
	"https://rpc.ankr.com/avalanche",
	"https://avalanche.public-rpc.com",
	"https://avalanche.drpc.org",
	"https://rpc.tornadoeth.cash/avax",
	"https://api.zan.top/node/v1/avax/mainnet/public/ext/bc/C/rpc",
	"https://1rpc.io/avax/c",
	"https://endpoints.omniatech.io/v1/avax/mainnet/public",
	"https://blastapi.io/public-api/avalanche",
	"https://avalancheapi.terminet.io/ext/bc/C/rpc",
	"https://avax-pokt.nodies.app/ext/bc/C/rpc",
	"https://avalanche.api.onfinality.io/public/ext/bc/C/rpc",
	"https://avalanche-c-chain-rpc.publicnode.com",
}

func main() {
	if err := godotenv.Load(); err != nil {
		log.Fatal("Error loading .env file")
	}

	privateKeyHex, numWorkersStr, transactionsNumberStr, jsonData := getConfig()

	var rpcIndex int
	var client *ethclient.Client
	var err error
	for {
		client, err = ethclient.Dial(rpcURLs[rpcIndex])
		if err != nil {
			log.Printf("Error connecting to eth client at %s: %v", rpcURLs[rpcIndex], err)
			rpcIndex = (rpcIndex + 1) % len(rpcURLs)
			continue
		}
		break
	}

	privateKey, fromAddress := ethereumSetup(privateKeyHex, client)

	nonce := getInitialNonce(client, fromAddress)

	txnsChan := make(chan int, transactionsNumberStr)
	defer close(txnsChan)

	gasPrice := getGasPrice(client)

	var wg sync.WaitGroup
	for i := 0; i < numWorkersStr; i++ {
		wg.Add(1)
		go worker(&wg, txnsChan, jsonData, nonce, gasPrice, privateKey, fromAddress, &rpcIndex)
		nonce += uint64(transactionsNumberStr) / uint64(numWorkersStr)
	}

	for i := 0; i < transactionsNumberStr; i++ {
		txnsChan <- i
		time.Sleep(500 * time.Millisecond) 
	}

	wg.Wait()
	fmt.Println("✨ All transactions sent.")
}

func getConfig() (privateKeyHex string, numWorkersStr, transactionsNumberStr int, jsonData string) {
	privateKeyHex = os.Getenv("PRIVATE_KEY_HEX")
	numWorkersStr, _ = strconv.Atoi(os.Getenv("NUM_WORKERS"))
	transactionsNumberStr, _ = strconv.Atoi(os.Getenv("TRANSACTIONS_NUMBER"))
	jsonData = os.Getenv("JSON_DATA")
	return
}

func ethereumSetup(privateKeyHex string, client *ethclient.Client) (*ecdsa.PrivateKey, common.Address) {
	privateKey, err := crypto.HexToECDSA(privateKeyHex)
	if err != nil {
		log.Fatal("error reading private key:", err)
	}

	publicKeyECDSA, ok := privateKey.Public().(*ecdsa.PublicKey)
	if !ok {
		log.Fatal("error casting public key to ECDSA")
	}

	fromAddress := crypto.PubkeyToAddress(*publicKeyECDSA)
	return privateKey, fromAddress
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

func worker(wg *sync.WaitGroup, txnsChan <-chan int, jsonData string, startNonce uint64, gasPrice *big.Int, privateKey *ecdsa.PrivateKey, fromAddress common.Address, rpcIndex *int) {
	defer wg.Done()

	nonce := startNonce
	txCount := 0
	for range txnsChan {
		client, err := ethclient.Dial(rpcURLs[*rpcIndex])
		if err != nil {
			log.Printf("Error connecting to eth client at %s: %v", rpcURLs[*rpcIndex], err)
			*rpcIndex = (*rpcIndex + 1) % len(rpcURLs)
			fmt.Println("Changing rpc...")
			continue
		}

		if err := sendTransaction(client, privateKey, fromAddress, jsonData, nonce, gasPrice, rpcIndex); err != nil {
			fmt.Println("Changing rpc...")
			*rpcIndex = (*rpcIndex + 1) % len(rpcURLs)
			client, _ = ethclient.Dial(rpcURLs[*rpcIndex]) 
			sendTransaction(client, privateKey, fromAddress, jsonData, nonce, gasPrice, rpcIndex)
		}
		nonce++
		txCount++
	}
}

func sendTransaction(client *ethclient.Client, privateKey *ecdsa.PrivateKey, fromAddress common.Address, jsonData string, nonce uint64, gasPrice *big.Int, rpcIndex *int) error {
	value := big.NewInt(0)
	gasLimit := uint64(22000)
	data := []byte(jsonData)

	tx := types.NewTransaction(nonce, fromAddress, value, gasLimit, gasPrice, data)

	chainID, err := client.NetworkID(context.Background())
	if err != nil {
		log.Printf("error getting network ID: %v", err)
		return err
	}

	signedTx, err := types.SignTx(tx, types.NewEIP155Signer(chainID), privateKey)
	if err != nil {
		log.Printf("error signing transaction: %v", err)
		return err
	}

	if err := client.SendTransaction(context.Background(), signedTx); err != nil {
		log.Printf("error sending transaction: %v", err)
		if strings.Contains(err.Error(), "429") {
			return err
		}
		return err
	}

	fmt.Printf("📜 Transaction sent: %s\n", signedTx.Hash().Hex())
	return nil
}

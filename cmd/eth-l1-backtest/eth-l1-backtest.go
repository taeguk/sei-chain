package main

import (
	"context"
	"fmt"
	"log"
	"math/big"
	"os"
	"time"

	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/joho/godotenv"
)

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Fatalf("Error loading .env file: %v", err)
	}

	// connect to alchemy to pull txns from
	alchemy_api_key := os.Getenv("ALCHEMY_API_KEY")
	rpc_endpoint := fmt.Sprintf("https://eth-mainnet.g.alchemy.com/v2/%s", alchemy_api_key)
	client, err := ethclient.Dial(rpc_endpoint)
	if err != nil {
		log.Fatalf("Failed to connect to the Ethereum client: %v", err)
	}

	// connect to (local) sei evm to send txns to
	local_evm_endpoint := "http://127.0.0.1:8545"
	local_evm, err := ethclient.Dial(local_evm_endpoint)
	if err != nil {
		log.Fatalf("Failed to connect to local evm endpoint %v", err)
	}

	// pull txns from first ethereum block with some transactions
	// Start from block number 1 (you can adjust this as needed)
	// first eth transaction with at least 1 eth txn was block 46147
	var startBlockNumber int64 = 46147 - 1

	// Use a loop to iterate through the blocks
	fmt.Println("Starting to iterate through blocks...")
	for {
		blockNumber := big.NewInt(startBlockNumber)
		block, err := client.BlockByNumber(context.Background(), blockNumber)
		if err != nil {
			log.Printf("Failed to get block %d: %v", startBlockNumber, err)
			time.Sleep(1 * time.Second) // Wait a second before retrying
			continue
		}

		fmt.Printf("Number of transactions in block %d, block: %d\n", block.Number(), len(block.Transactions()))
		for _, tx := range block.Transactions() {
			fmt.Printf("Tx Hash: %s\n", tx.Hash().Hex())
			// send transaction to local_evm
			err = local_evm.SendTransaction(context.Background(), tx)
			if err != nil {
				log.Fatalf("Failed to send transaction to local evm %v", err)
			}
		}

		startBlockNumber++                 // Move to the next block
		time.Sleep(500 * time.Millisecond) // Wait a bit before fetching the next block
	}
}

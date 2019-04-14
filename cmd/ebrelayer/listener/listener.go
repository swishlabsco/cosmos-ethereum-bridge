package listener

import (
    "context"
    "fmt"
    "log"
    "math/big"

    "github.com/ethereum/go-ethereum"
    "github.com/ethereum/go-ethereum/crypto"
    "github.com/ethereum/go-ethereum/common"
    "github.com/ethereum/go-ethereum/core/types"
    "github.com/ethereum/go-ethereum/ethclient"
)

type LogLock struct {
    TransactionHash        string
    EthereumSender         common.Address
    CosmosRecipient        common.Address
    Value                  *big.Int
}

// -------------------------------------------------------------------------
// Starts an event listener on a specific network, contract, and event
// -------------------------------------------------------------------------
func start() {
    // Start client with infura ropsten provider
    // TODO: provider support for different chains with 'FlagNetwork'
    client, err := ethclient.Dial("wss://ropsten.infura.io/ws")
    if err != nil {
        log.Fatal(err)
    }

    // Deployed contract address and event signature
    // TODO: different multiple contracts with 'FlagContract'
    contractAddress := common.HexToAddress("0xe56143b75f4eeac5fa80dc6ffd912d4a3ed21fdf")
    // TODO: support multiple events with 'FlagEventSig'
    logLockSig := []byte("LogLock(address,address,uint256)")
    logLockEvent := crypto.Keccak256Hash(logLockSig)

    // Filter currently captures all events from the contract
    query := ethereum.FilterQuery{
        Addresses: []common.Address{contractAddress},
    }

    logs := make(chan types.Log)

    // Subscribe to the client, filter based on query, write events to logs
    sub, err := client.SubscribeFilterLogs(context.Background(), query, logs)
    if err != nil {
        log.Fatal(err)
    }

    for {
        select {
        case err := <-sub.Err():
            log.Fatal(err)
        case vLog := <-logs:
            fmt.Println("\nBlock Number:", vLog.BlockNumber)

            // Check if the event is a 'LogLock' event
            if vLog.Topics[0].Hex() == logLockEvent.Hex() {

                var lockEvent LogLock

                // Populate LogLock with event information
                lockEvent.TransactionHash = vLog.TxHash.String()
                lockEvent.EthereumSender = common.HexToAddress(vLog.Topics[1].Hex())
                lockEvent.CosmosRecipient = common.HexToAddress(vLog.Topics[2].Hex())
                lockEvent.Value = vLog.Topics[3].Big()

                fmt.Printf("Tx Hash: %s\n", lockEvent.TransactionHash)
                fmt.Printf("Ethereum Sender: %s\n", lockEvent.EthereumSender.Hex())
                fmt.Printf("Cosmos Recipient: %s\n", lockEvent.CosmosRecipient.Hex())
                fmt.Printf("Amount: %d\n", lockEvent.Value)

                // TODO: Send lockEvent to Cosmos tx generator
                // txs.relay.relay(lockEvent)
            }
        }
    }
}
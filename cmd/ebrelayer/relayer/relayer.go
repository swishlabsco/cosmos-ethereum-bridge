package relayer

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
    // "github.com/swishlabsco/cosmos-ethereum-bridge/cmd/ebrelayer/txs"
)

type LogLock struct {
    TransactionHash        string
    EthereumSender         common.Address
    CosmosRecipient        common.Address
    Value                  *big.Int
    Nonce                  *big.Int
}

// -------------------------------------------------------------------------
// Starts an event listener on a specific network, contract, and event
// -------------------------------------------------------------------------
// Testing parameters:
//      chainId: "3"
//      provider: "wss://ropsten.infura.io/ws"
//      peggyContractAddress: "0xe56143b75f4eeac5fa80dc6ffd912d4a3ed21fdf"
//      eventSignature: "LogLock(address,address,uint256)"
//      validatorPrefix: "validator"
//      validatorPassword: "12345678"
//
func InitRelayer(
    chainId string,
    provider string,
    peggyContractAddress string,
    eventSignature string,
    validatorPrefix string,
    validatorPassword string
) {
    //Check chain ID
    if(chainId != "3") {
        return fmt.Errorf("Only the ropsten network is currently supported (chainId = 3")
    }

    // Start client with infura ropsten provider
     //TODO: Implement geth lcd
    client, err := ethclient.Dial(provider)
    if err != nil {
        log.Fatal(err)
    }

    // Deployed contract address and event signature
    contractAddress := common.HexToAddress(peggyContractAddress)
    logLockSig := []byte(eventSignature)
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
                lockEvent.Nonce = vLog.Topics[4].Big()

                fmt.Printf("Tx Hash: %s\n", lockEvent.TransactionHash)
                fmt.Printf("Ethereum Sender: %s\n", lockEvent.EthereumSender.Hex())
                fmt.Printf("Cosmos Recipient: %s\n", lockEvent.CosmosRecipient.Hex())
                fmt.Printf("Amount: %d\n", lockEvent.Value)
                fmt.Printf("Nonce: %d\n", lockEvent.Nonce)

                txs.sendRelayTx(
                    "genesis-alpha",
                     validatorPrefix,
                     validatorPassword,
                     lockEvent.EthereumSender.Hex(),
                     lockEvent.CosmosRecipient.Hex(),
                     lockEvent.Value,
                     lockEvent.Nonce)
            }
        }
    }
}

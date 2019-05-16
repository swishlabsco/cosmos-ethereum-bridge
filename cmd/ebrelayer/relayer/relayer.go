package relayer

import (
    "context"
    "log"
    "encoding/hex"
    "fmt"
    "math/big"
    
    sdk "github.com/cosmos/cosmos-sdk/types"

    "github.com/ethereum/go-ethereum/common"
    "github.com/ethereum/go-ethereum"
    "github.com/ethereum/go-ethereum/crypto"
    "github.com/ethereum/go-ethereum/core/types"
    "github.com/ethereum/go-ethereum/ethclient"

    // "github.com/swishlabsco/cosmos-ethereum-bridge/cmd/ebrelayer/txs"
    "github.com/cosmos/cosmos-sdk/codec"

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
//    validator = sdk.AccAddress("cosmos1xdp5tvt7lxh8rf9xx07wy2xlagzhq24ha48xtq")
//    chainId = "testing"
//    ethereumProvider = "wss://ropsten.infura.io/ws"
//    peggyContractAddress = "0xe56143b75f4eeac5fa80dc6ffd912d4a3ed21fdf"
//    eventSignature = "LogLock(address,address,uint256)"

func InitRelayer(
    cdc *codec.Codec,
    chainId string,
    provider string,
    peggyContractAddress string,
    eventSignature string,
    validator sdk.AccAddress) error {

    // Console log for testing purposes...
    fmt.Printf("\n\ninitRelayer() received params:\n")
    fmt.Printf("chainId: %s\n", chainId)
    fmt.Printf("provider: %s\n", provider)
    fmt.Printf("peggyContractAddress: %s\n", peggyContractAddress)
    fmt.Printf("eventSignature: %s\n", eventSignature)
    fmt.Printf("validator: %s\n\n", validator)

   // Start client with infura ropsten provider
    client, err := ethclient.Dial(provider)
    if err != nil {
        log.Fatal(err)
    }

    // Deployed contract address and event signature
    b, err := hex.DecodeString(peggyContractAddress)
    if err != nil{
        return fmt.Errorf("Error while decoding contract address")
    }

    contractAddress := common.HexToAddress(peggyContractAddress)
    logLockSig := []byte(eventSignature)
    logLockEvent := crypto.Keccak256Hash(logLockSig)

    fmt.Printf("\n\nContract Address: %s\n Log Lock Signature: %s\n\n",
                b, logLockSig)

    fmt.Printf("%s", logLockEvent)
    

    // TODO: resolve type casting error between go-ethereum/common and swish/go-ethereum/common
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

                fmt.Printf("Event Message parameters:\n Cosmos Recipient: %s,\n Validator: %s,\n Nonce: %d,\n Ethereum Address: %s,\n Amount: %s\n",
                            lockEvent.CosmosRecipient, validator, lockEvent.Nonce, lockEvent.EthereumSender, lockEvent.Value)

                // txs.RelayEvent(cdc, "cosmos1xdpfh8vt7lxh8rf9xx0pjzy2xlagzhq24ha48xtq", validator, 0, "0xe88843b75f4eeaj5fa80dc6ffd912d4a3ed21lz4", 1001)
                
            }
        }
    }
    return fmt.Errorf("Error: Relayer timed out.")
}

// TODO: This is an attempt to solve bug on line 80.
//       Remove this once resolved
// func create(addr common.Address) *common.Address {
//     return &addr
// }

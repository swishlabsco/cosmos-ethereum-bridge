package relayer

import (
    "context"
    "log"
    "strconv"
    "fmt"
    "math/big"
    
    sdk "github.com/cosmos/cosmos-sdk/types"

    "github.com/ethereum/go-ethereum/common"
    "github.com/ethereum/go-ethereum"
    "github.com/ethereum/go-ethereum/crypto"
    "github.com/ethereum/go-ethereum/core/types"
    "github.com/ethereum/go-ethereum/ethclient"

    "github.com/swishlabsco/cosmos-ethereum-bridge/cmd/ebrelayer/txs"

    // TODO: Re-integrate codec as parameter of InitRelayer(...)
    // "github.com/cosmos/cosmos-sdk/codec"

)

type LogLock struct {
    TransactionHash        string
    EthereumSender         common.Address
    CosmosRecipient        sdk.AccAddress
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
    // cdc *codec.Codec,
    chainId string,
    provider string,
    peggyContractAddress string,
    eventSignature string,
    validator sdk.AccAddress) error {

    // Console log for testing purposes...
    fmt.Printf("initRelayer() received params:\n")
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
    contractAddress := common.HexToAddress(peggyContractAddress)
    logLockSig := []byte(eventSignature)
    logLockEvent := crypto.Keccak256Hash(logLockSig)

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

                // TODO: remove printing
                fmt.Printf("Tx Hash: %s\n", lockEvent.TransactionHash)
                fmt.Printf("Ethereum Sender: %s\n", lockEvent.EthereumSender.Hex())
                fmt.Printf("Cosmos Recipient: %s\n", lockEvent.CosmosRecipient.Hex())
                fmt.Printf("Amount: %d\n", lockEvent.Value)
                fmt.Printf("Nonce: %d\n", lockEvent.Nonce)

                // TODO: get validator certification files from the current validator
                const activeValidator, err2 := sdk.AccAddressFromBech32(activeValidator)
                if err2 != nil {
                    fmt.Errorf("%s", err2)
                }

                //Get cosmos recipient
                const cosmosRecipient, err1 := sdk.AccAddressFromBech32(lockEvent.CosmosRecipient.Hex())
                if err1 != nil {
                    fmt.Errorf("%s", err1)
                }

                // TODO: Add token address to lockEvent struct for token parsing,
                //       if field == '0x00000000....' then use string 'ethereum'
                //       for decoding
                
                //Get coin amount and correct for wei 10**18
                amount, err3 := sdk.ParseCoins(strings,Join((strconv.Itoa(lockEvent.Value/(Pow(10.0, 18), "ethereum"))
                if err3 != nil {
                    fmt.Errorf("%s", err3)
                }

                fmt.Printf("Event Message parameters:\n Cosmos Recipient: %s,\n Validator: %s,\n Nonce: %d,\n Ethereum Address: %s,\n Amount: %s\n",
                            cosmosRecipient, activeValidator, lockEvent.Nonce, lockEvent.Value, amount)

                relay.relayEvent(cdc, cosmosRecipient, activeValidator, lockEvent.Nonce, lockEvent.Value, amount)
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

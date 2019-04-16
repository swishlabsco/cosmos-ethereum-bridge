package txs

import (
  "fmt"
  "math/rand"
  "runtime"
  "strings"
  "sync"
  "time"

  "github.com/cosmos/cosmos-sdk/client/context"
  "github.com/cosmos/cosmos-sdk/client/keys"
  cryptokeys "github.com/cosmos/cosmos-sdk/crypto/keys"
  sdk "github.com/cosmos/cosmos-sdk/types"
  "github.com/cosmos/cosmos-sdk/wire"
  authcmd "github.com/cosmos/cosmos-sdk/x/auth/client/cli"
  authctx "github.com/cosmos/cosmos-sdk/x/auth/client/context"
  "github.com/cosmos/cosmos-sdk/x/bank/client"
  "github.com/spf13/cobra"
  "github.com/spf13/viper"
  tmcrypto "github.com/tendermint/tendermint/crypto"
  "github.com/swishlabsco/cosmos-ethereum-bridge/cmd/ebrelayer/helpers"
  "github.com/swishlabsco/cosmos-ethereum-bridge/cmd/ebrelayer/log"
  "github.com/swishlabsco/cosmos-ethereum-bridge/cmd/ebrelayer/stats"
  "github.com/swishlabsco/cosmos-ethereum-bridge/x/oracle"
)

var (
  defaultBlockTime time.Duration = 1000
)

type relayCtx struct {
  chainID         string
  stats           *stats.Stats
  ethereumSender  string
  cosmosReceiver  sdk.AccAddress
  amount          sdk.Coins
}

// -------------------------------------------------------------------------
// Parses cmd line arguments, constructs relay context, requests validator
// creation and once available, relays the transaction to the oracle. 
// -------------------------------------------------------------------------
func initRelayer(cdc *wire.Codec) func(args []string) error {
  // Parse chain's ID
  chainID := args[0]
  if chainID == "" {
    return fmt.Errorf("Must specify chain id")
  }

  // Parse validator prefix
  validatorPrefix := args[1]

  //TODO: Sanitize input
  // Parse validator password
  validatorPassword := args[2]
  if validatorPassword == "" {
    return fmt.Errorf("Must specify validator password")
  }

  // Parse ethereum sender
  ethereumSender := args[3]
  if ethereumSender == "" {
    err = fmt.Errorf("Invalid ethereum sender")
    return
  }

  // Parse the cosmos receiver
  cosmosReceiver := args[4]
  if ethereumSender == "" {
    err = fmt.Errorf("Invalid Cosmos receiver")
    return
  }

  // Parse the transaction amount
  amount := args[5]
  if amount == "" {
    err = fmt.Errorf("Must specify the amount locked")
    return
  }

  // Create new stats
  stats := stats.NewStats()

  // Construct the relay context
  relayCtx := relayCtx{chainID, &stats, ethereumSender, cosmosReceiver, amount}

  // Create validator thread
  accountName := info.GetName()
  validator Validator; 
  if strings.HasPrefix(accountName, validatorPrefix) {
    validator, err = spawnValidator(relayCtx, cdc, validatorPassword, kb, info, &validator)
  } else {
    log.Log.Warningf("Error while spawning validator %v\n", err)
    relayCtx.stats.AddError()
    return;
  }

  // Prints current relay stats at regular intervals
  go doEvery(1*time.Second, relayCtx.stats.Print)

  // Validator attempts to relay this transaction
  go validators[i].relay(&relayCtx)
}


// -------------------------------------------------------------------------
// Applies time delay to go routine
// -------------------------------------------------------------------------
func doEvery(d time.Duration, f func()) {
  for range time.Tick(d) {
    go f()
  }
}

// -------------------------------------------------------------------------
// All the things needed for a single validator thread
// -------------------------------------------------------------------------
type Validator struct {
  accountName    string
  password       string
  accountAddress sdk.AccAddress
  cdc            *wire.Codec
  nextSequence   int64
  cliCtx         context.CLIContext
  txCtx          authctx.TxContext
  priv           tmcrypto.PrivKey
  currentCoins   sdk.Coins
  sequenceCheck  int
  queryFree      chan bool
}

// -------------------------------------------------------------------------
// This function builds, signs, and broadcasts txs for a single validator
// -------------------------------------------------------------------------
func (vl *Validator) relay(relayCtx *relayCtx) {
  <-vl.queryFree

  log.Log.Debugf("Validator attempting to relay with sequence %v...\n", vl.nextSequence)

  // Make a bridge claim using the context
  var msg sdk.Msg
  var ok bool
  msg, ok = vl.makeBridgeClaim(relayCtx)

  // If msg construction returned an error, return
  if !ok {
    log.Log.Warningf("Validator received error while making bridge claim: %v\n", err)
    relayCtx.stats.AddError()
    vl.updateSequence()
    vl.queryFree <- true
    return
  }

  // Get the transaction context at validator's current sequence
  vl.txCtx = vl.txCtx.WithSequence(vl.nextSequence)

  // Build transaction, sign with private key, and broadcast to network
  _, err := helpers.PrivBuildSignAndBroadcastMsg(vl.cdc, vl.cliCtx, vl.txCtx, vl.priv, msg)

  // Increment sequence/nonce
  vl.nextSequence = vl.nextSequence + 1
  vl.sequenceCheck = vl.sequenceCheck + 1

  if err != nil {
    log.Log.Warningf("Validator received error trying to relay: %v\n", err)
    relayCtx.stats.AddError()
    vl.updateSequence()
    vl.queryFree <- true
    return
  }

  log.Log.Debugf("Validator sending successful\n")
  relayCtx.stats.AddSuccess()

  vl.queryFree <- true
}

// -------------------------------------------------------------------------
// Builds a validator's BridgeClaim msg
// -------------------------------------------------------------------------
func (vl *Validator) makeBridgeClaim(relayCtx *relayCtx) (sdk.Msg, bool) {

  // Declare BridgeClaim attributes
  var nonce int64
  var ethereumSender string
  var cosmosReceiver sdk.AccAddress
  var validator sdk.AccAddress
  var amount sdk.Coin


  // Assign values to BridgeClaim attributes
  nonce = vl.nonce
  ethereumSender = relayCtx.ethereumSender
  cosmosReceiver = relayCtx.cosmosReceiver
  validator = vl.accountAddress
  amount = sdk.NewCoin(relayCtx.amount)


  // Create a new bridge claim with these attributes
  msg := oracle.NewMsgMakeBridgeClaim(
    nonce, ethereumSender, cosmosReceiver validator, amount)

  log.Log.Debugf("Validator %v: made claim of %v->%v an amount of %v\\n",
    vl.index, ethereumSender, cosmosReceiver, amount)

  return msg, true
}

// -------------------------------------------------------------------------
// This function will be used to update the validator's nonce, inspired
// by 'updateSequenceAndCoins()'
// -------------------------------------------------------------------------
func (vl *Validator) updateSequence() {
  log.Log.Debugf("Validator %v: Time to refresh sequence and coins, waiting for next block...\n", vl.index)
  time.Sleep(defaultBlockTime * time.Millisecond)

  log.Log.Debugf("Validator %v: Querying account for new sequence and coins...\n", vl.index)
  fromAcc, err := vl.cliCtx.GetAccount(vl.accountAddress)
  if err != nil {
    log.Log.Errorf("Validator %v: Account not found, skipping\n", vl.index)
    return
  }

  sequence, err := vl.cliCtx.GetAccountSequence(vl.accountAddress)
  if err != nil {
    log.Log.Errorf("Validator %v: Error getting sequence: %v\n", vl.index, err)
  }
  vl.nextSequence = sequence
  log.Log.Debugf("Validator %v: Sequence updated to %v\n", vl.index, vl.nextSequence)
  vl.sequenceCheck = 0
}

// -------------------------------------------------------------------------
// Creates an individual validator account
// -------------------------------------------------------------------------
func (vl *Validator) spawnValidator(
  relayCtx *relayCtx, cdc *wire.Codec, validatorPassword string,
  kb cryptokeys.Keybase, info cryptokeys.Info) {

  log.Log.Debugf("Spawning a validator...")

  log.Log.Debugf("Making contexts...\n")

  cliCtx := context.NewCLIContext().
    WithCodec(cdc).
    WithAccountDecoder(authcmd.GetAccountDecoder(cdc)).
    WithFromAddressName(info.GetName())

  txCtx := authctx.TxContext{
    Codec:   cdc,
    Gas:     20000,
    ChainID: spamCtx.chainID,
  }

  log.Log.Debugf("Validating account...\n")

  address := sdk.AccAddress(info.GetPubKey().Address())
  account, err3 := cliCtx.GetAccount(address)
  if err3 != nil {
    log.Log.Errorf("Validator account address check failed: %s\n", err3)
    return
  }

  txCtx = txCtx.WithAccountNumber(account.GetAccountNumber())
  txCtx = txCtx.WithSequence(account.GetSequence())

  // get private key
  priv, err := kb.ExportPrivateKeyObject(info.GetName(), validatorPassword)
  if err != nil {
    panic(err)
  }

  newValidator := Validator{
    info.GetName(), validatorPassword, address, cdc, account.GetSequence(),
    cliCtx, txCtx, priv, account.GetCoins(), 0, queryFree,
  }

  log.Log.Infof("Validator %s spawned...\n", address)

  return newValidator
}

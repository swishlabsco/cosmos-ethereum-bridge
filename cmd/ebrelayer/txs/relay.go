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
func GetRelayCmd(cdc *wire.Codec) func(cmd *cobra.Command, args []string) error {
  // Parse chain's ID
  return func(_ *cobra.Command, _ []string) error {
    chainID := viper.GetString(FlagChainID)
    if chainID == "" {
      return fmt.Errorf("--chain-id is required")
    }

    // Parse rate limit
    rateLimit := viper.GetFloat64(FlagRateLimit)

    // Parse validator prefix and password
    validatorPrefix := viper.GetString(FlagValidatorPrefix)
    validatorPassword := viper.GetString(FlagValidatorPassword)
    if validatorPassword == "" {
      return fmt.Errorf("--relay-password is required")
    }

    stats := stats.NewStats()

    // Parse the ethereum sender
    ethereumSender, err := viper.GetString(FlagSender)
    if err != nil {
      return err
    }
    if ethereumSender == "" {
      err = fmt.Errorf("--sender is required")
      return
    }

    // Parse the cosmos receiver
    cosmosReceiver, err := viper.GetString(FlagReceiver)
    if err != nil {
      return err
    }
    if cosmosReceiver == "" {
      err = fmt.Errorf("--receiver is required")
      return
    }

    // Parse the transaction amount
    amount, err := viper.GetFloat64(FlagAmount)
    if err != nil {
      return err
    }
    if amount == "" {
      err = fmt.Errorf("--amount is required")
      return
    }

    // Construct the relay context
    relayCtx := relayCtx{chainID, &stats, ethereumSender, cosmosReceiver, amount}

    // Create context and all validator objects
    validators, err := createValidators(&relayCtx, cdc, validatorPrefix, validatorPassword)
    if err != nil {
      return err
    }

    // Must have at least 1 available validator
    if len(validators) == 0 {
      return fmt.Errorf("no validators are online")
    }

    log.Log.Infof("Found %v validator accounts\n", len(validators))

    // Use all cores
    runtime.GOMAXPROCS(runtime.NumCPU())

    // Rate limiter to allow x events per second
    limiter := time.Tick(time.Duration(rateLimit) * time.Millisecond)

    // Go routine which prints current relay status
    go doEvery(1*time.Second, relayCtx.stats.Print)

    i := 0

    // Go routine where each validator relays the transaction 
    for {
      <-limiter
      nextValidator := validators[(i+1)%len(validators)]
      go validators[i].relay(&relayCtx, &nextValidator)
      if i == len(validators)-1 {
        i = 0
      } else {
        i++
      }
    }
  }
}

// -------------------------------------------------------------------------
// Applies the specified time delay to a go routine
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
  index          int
  nextSequence   int64
  nonce          int //TODO: apply nonce
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
func (vl *Validator) relay(relayCtx *relayCtx, nextValidator *Validator) {
  <-vl.queryFree

  log.Log.Debugf("Validator %v: relay with sequence %v...\n", vl.index, vl.nextSequence)

  var msg sdk.Msg
  var ok bool

  msg, ok = vl.makeBridgeClaim(relayCtx)

  // If msg construction returned an error, move to the next validator
  if !ok {
    vl.updateSequence()
    vl.queryFree <- true
    return
  }

  vl.txCtx = vl.txCtx.WithSequence(vl.nextSequence)

  _, err := helpers.PrivBuildSignAndBroadcastMsg(vl.cdc, vl.cliCtx, vl.txCtx, vl.priv, msg)

  vl.nextSequence = vl.nextSequence + 1
  vl.sequenceCheck = vl.sequenceCheck + 1

  if err != nil {
    log.Log.Warningf("Validator %v: Received error trying to relay: %v\n", vl.index, err)
    relayCtx.stats.AddError()
    vl.updateSequence()
    vl.queryFree <- true
    return
  }
  log.Log.Debugf("Validator %v: Sending successful\n", vl.index)
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
// Starts multiple validator go routines
// -------------------------------------------------------------------------
func createValidators(relayCtx *relayCtx, cdc *wire.Codec, validatorPrefix string, validatorPassword string) ([]Validator, error) {
  kb, err := keys.GetKeyBase()
  if err != nil {
    return nil, err
  }

  infos, err := kb.List()
  if err != nil {
    return nil, err
  }

  var wg sync.WaitGroup
  semaphore := make(chan bool, 50)
  var validators []Validator
  var j = -1
  for _, info := range infos {
    accountName := info.GetName()
    if strings.HasPrefix(accountName, validatorPrefix) {
      j++
      wg.Add(1)
      semaphore <- true
      go spawnValidator(relayCtx, cdc, validatorPassword, j, kb, info, &validator, &wg, semaphore)
    }
  }

  wg.Wait()

  return validators, nil
}
// -------------------------------------------------------------------------
// Creates an individual validator account and adds it to the validators
// array, printing each step of the process to the console
// -------------------------------------------------------------------------
func spawnValidator(relayCtx *relayCtx, cdc *wire.Codec, validatorPassword string, index int, kb cryptokeys.Keybase,
  info cryptokeys.Info, validators *[]Validator, wg *sync.WaitGroup,
  semaphore <-chan bool) {
  defer wg.Done()

  log.Log.Debugf("Validator %v: Spawning...\n", index)

  log.Log.Debugf("Validator %v: Making contexts...\n", index)

  cliCtx := context.NewCLIContext().
    WithCodec(cdc).
    WithAccountDecoder(authcmd.GetAccountDecoder(cdc)).
    WithFromAddressName(info.GetName())

  txCtx := authctx.TxContext{
    Codec:   cdc,
    Gas:     20000,
    ChainID: spamCtx.chainID,
  }

  log.Log.Debugf("Validator %v: Finding account...\n", index)

  address := sdk.AccAddress(info.GetPubKey().Address())
  account, err3 := cliCtx.GetAccount(address)
  if err3 != nil {
    log.Log.Errorf("Validator %v: Account not found, skipping\n", index)
    <-semaphore
    return
  }

  txCtx = txCtx.WithAccountNumber(account.GetAccountNumber())
  txCtx = txCtx.WithSequence(account.GetSequence())

  // get private key
  priv, err := kb.ExportPrivateKeyObject(info.GetName(), validatorPassword)
  if err != nil {
    panic(err)
  }

  queryFree := make(chan bool, 1)
  queryFree <- true

  newValidator := Validator{
    info.GetName(), validatorPassword, address, cdc, index, account.GetSequence(), cliCtx, txCtx, priv,
    account.GetCoins(), 0, queryFree,
  }

  *validators = append(*validators, newValidator)
  log.Log.Infof("Validator %v: Spawned...\n", index)
  <-semaphore
}

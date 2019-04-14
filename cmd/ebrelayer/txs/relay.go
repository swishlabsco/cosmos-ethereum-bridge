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

// TODO: Update to hold relevant information
type relayCtx struct {
  chainID         string
  stats           *stats.Stats
}

// -------------------------------------------------------------------------
// Returns the command to relay the transaction to the oracle
// -------------------------------------------------------------------------
func GetRelayCmd(cdc *wire.Codec) func(cmd *cobra.Command, args []string) error {
  return func(_ *cobra.Command, _ []string) error {
    chainID := viper.GetString(FlagChainID)
    if chainID == "" {
      return fmt.Errorf("--chain-id is required")
    }

    rateLimit := viper.GetFloat64(FlagRateLimit)

    // parse validator prefix and password
    validatorPrefix := viper.GetString(FlagValidatorPrefix)
    validatorPassword := viper.GetString(FlagValidatorPassword)
    if validatorPassword == "" {
      return fmt.Errorf("--relay-password is required")
    }

    stats := stats.NewStats()

    relayCtx := relayCtx{chainID, &stats}

    // create context and all validator objects
    validators, err := createValidators(&relayCtx, cdc, validatorPrefix, validatorPassword)
    if err != nil {
      return err
    }

    // ensure at least 1 validator is present
    if len(validators) == 0 {
      return fmt.Errorf("no validators are online")
    }

    log.Log.Infof("Found %v validator accounts\n", len(validators))

    //Use all cores
    runtime.GOMAXPROCS(runtime.NumCPU())

    // rate limiter to allow x events per second
    limiter := time.Tick(time.Duration(rateLimit) * time.Millisecond)

    go doEvery(1*time.Second, relayCtx.stats.Print)

    i := 0

    for {
      <-limiter
      // -------------------------------------------------------------------------
      // TODO: This is the primary validator relay go routine, it must be correct!
      // -------------------------------------------------------------------------
      // nextValidator := validators[(i+1)%len(validators)]
      // go validators[i].relay(&relayCtx, &nextValidator)
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

//All the things needed for a single validator thread
type Validator struct {
  accountName    string
  password       string
  accountAddress sdk.AccAddress
  cdc            *wire.Codec
  index          int
  nextSequence   int64
  cliCtx         context.CLIContext
  txCtx          authctx.TxContext
  priv           tmcrypto.PrivKey
  currentCoins   sdk.Coins
  sequenceCheck  int // TODO: update to fulfill nonce functionality
  queryFree      chan bool
}

// -------------------------------------------------------------------------
// This function builds, signs, and broadcasts txs for a single validator
// -------------------------------------------------------------------------
func (vl *Validator) relay(relayCtx *relayCtx, oracle *Validator) { //TODO: update oracle datatype
  <-vl.queryFree

  log.Log.Debugf("Validator %v: relay with sequence %v...\n", vl.index, vl.nextSequence)

  var msg sdk.Msg
  var ok bool

  msg, coinsUsed, ok = vl.makeTxMsg(relayCtx)

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

  vl.currentCoins = vl.currentCoins.Minus(coinsUsed)
  if vl.sequenceCheck >= 200 {
    vl.updateSequence()
  }
  vl.queryFree <- true
}

// -------------------------------------------------------------------------
// This function builds, signs, and broadcasts txs for a single validator
// -------------------------------------------------------------------------
func (vl *Validator) makeTxMsg(relayCtx *relayCtx) (sdk.Msg, bool) {

  //TODO: formulate the relay message; should relayCtx should this information?
  msg := client.NewRelayMsg(vl.nextSequence, relayCtx.ethereumSender, relayCtx.cosmosRecipient, relayCtx.amount, vl.index)

  log.Log.Debugf("Validator %v: Will relay: %v %v -> %v\n", sp.index, tx.ethSender, tx.cosmosRecipient, tx.amount)

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
// As validators are independent entities, the relayer shouldn't be able to 
// arbitrarily create validators which can be used to sign off on relays.
// As such, this function is likely outdated and only for testing.
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
      go SpawnValidator(relayCtx, cdc, validatorPassword, j, kb, info, &validator, &wg, semaphore)
    }
  }

  wg.Wait()

  return validators, nil
}
// -------------------------------------------------------------------------
// TODO: See 'createValidators()' function definition above
// -------------------------------------------------------------------------
// func SpawnValidator(spamCtx *spamCtx, cdc *wire.Codec, spamPassword string, index int, kb cryptokeys.Keybase,
//   info cryptokeys.Info, spammers *[]Spammer, wg *sync.WaitGroup,
//   semaphore <-chan bool) {
//   defer wg.Done()

//   log.Log.Debugf("Spammer %v: Spawning...\n", index)

//   log.Log.Debugf("Spammer %v: Making contexts...\n", index)

//   cliCtx := context.NewCLIContext().
//     WithCodec(cdc).
//     WithAccountDecoder(authcmd.GetAccountDecoder(cdc)).
//     WithFromAddressName(info.GetName())

//   txCtx := authctx.TxContext{
//     Codec:   cdc,
//     Gas:     20000,
//     ChainID: spamCtx.chainID,
//   }

//   log.Log.Debugf("Spammer %v: Finding account...\n", index)

//   address := sdk.AccAddress(info.GetPubKey().Address())
//   account, err3 := cliCtx.GetAccount(address)
//   if err3 != nil {
//     log.Log.Errorf("Spammer %v: Account not found, skipping\n", index)
//     <-semaphore
//     return
//   }

//   txCtx = txCtx.WithAccountNumber(account.GetAccountNumber())
//   txCtx = txCtx.WithSequence(account.GetSequence())

//   // get private key
//   priv, err := kb.ExportPrivateKeyObject(info.GetName(), spamPassword)
//   if err != nil {
//     panic(err)
//   }

//   queryFree := make(chan bool, 1)
//   queryFree <- true

//   newSpammer := Spammer{
//     info.GetName(), spamPassword, address, cdc, index, account.GetSequence(), cliCtx, txCtx, priv,
//     account.GetCoins(), 0, queryFree,
//   }

//   *spammers = append(*spammers, newSpammer)
//   log.Log.Infof("Spammer %v: Spawned...\n", index)
//   <-semaphore
// }

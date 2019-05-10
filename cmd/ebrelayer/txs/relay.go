package txs

import (
  "time"

  "github.com/cosmos/cosmos-sdk/codec"
  "github.com/cosmos/cosmos-sdk/client/context"
  tmcrypto "github.com/tendermint/tendermint/crypto"
  sdk "github.com/cosmos/cosmos-sdk/types"
  "github.com/swishlabsco/cosmos-ethereum-bridge/x/oracle"
  "github.com/swishlabsco/cosmos-ethereum-bridge/cmd/ebrelayer/log"
  
  // "github.com/cosmos/cosmos-sdk/client"
  // "github.com/cosmos/cosmos-sdk/server"
  // cryptokeys "github.com/cosmos/cosmos-sdk/crypto/keys"
  // bankcmd "github.com/cosmos/cosmos-sdk/x/bank/client/cli"
  // bank "github.com/cosmos/cosmos-sdk/x/bank/client/rest"
  // authtxb "github.com/cosmos/cosmos-sdk/x/bank/client/txbuilder"
  // authcmd "github.com/cosmos/cosmos-sdk/x/auth/client/cli"
  // auth "github.com/cosmos/cosmos-sdk/x/auth/client/rest"

)

var (
  defaultBlockTime time.Duration = 1000
)

type relayCtx struct {
  transactionHash string
  ethereumSender  string
  cosmosReceiver  sdk.AccAddress
  amount          sdk.Coins
  nonce           string
}

// -------------------------------------------------------------------------
// Parses cmd line arguments, constructs relay context, requests validator
// creation and once available, relays the transaction to the oracle. 
// -------------------------------------------------------------------------
func sendRelayTx(
    cdc                 *codec.Codec,
    validatorPrefix     string,
    validatorPassword   string,
    ethereumSender      string, 
    cosmosRecipient     string,
    value               string,
    nonce               string){

  // Construct the relay context
  relayTxCtx := relayCtx{ ethereumSender, cosmosRecipient, value, nonce}

  // Create validator thread
  // accountName := info.GetName()

  var validator Validator

  // TODO: Instead of spawning validators, use given validator account information
  // if strings.HasPrefix(accountName, validatorPrefix) {
  //   validator, err = spawnValidator(relayCtx, cdc, validatorPassword, kb, info, &validator)
  // } else {
  //   log.Log.Warningf("Error while spawning validator %v\n", err)
  //   relayCtx.stats.AddError()
  //   return;
  // }

  // Validator attempts to relay this transaction
  go validator.relay(&relayCtx)
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
  cdc            *codec.Codec
  nextSequence   int64
  cliCtx         context.CLIContext
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
    vl.queryFree <- true
    return
  }

  // Get the transaction context at validator's current sequence
  // vl.txCtx = vl.txCtx.WithSequence(vl.nextSequence)

  // Encode the transaction context with codec
  txCtx := EncodeTxRequestHandlerFn(vl.cdc, vl.cliCtx)

  // Build transaction, sign with private key, and broadcast to network
  handlerFunc = BroadcastTxRequest(vl.cdc, txCtx)

  // Increment sequence/nonce
  vl.nextSequence = vl.nextSequence + 1
  vl.sequenceCheck = vl.sequenceCheck + 1

  if err != nil {
    log.Log.Warningf("Validator received error trying to relay: %v\n", err)
    vl.queryFree <- true
    return
  }

  log.Log.Debugf("Validator sending successful\n")

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
    nonce, ethereumSender, cosmosReceiver, validator, amount)

  log.Log.Debugf("Validator %v: made claim of %v->%v an amount of %v\\n",
    vl.index, ethereumSender, cosmosReceiver, amount)

  return msg, true
}

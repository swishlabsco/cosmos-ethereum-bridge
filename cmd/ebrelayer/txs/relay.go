package txs

import(
  "fmt"

  "github.com/cosmos/cosmos-sdk/client/context"
  "github.com/cosmos/cosmos-sdk/client/utils"
  sdk "github.com/cosmos/cosmos-sdk/types"

  authtxb "github.com/cosmos/cosmos-sdk/x/auth/client/txbuilder"

  "github.com/swishlabsco/cosmos-ethereum-bridge/x/ethbridge/types"
  "github.com/cosmos/cosmos-sdk/codec"
)

type EthEvent struct {
    Nonce          int            `json:"nonce"`
    EthereumSender string         `json:"ethereum_sender"`
    CosmosRecipient sdk.AccAddress `json:"cosmos_receiver"`
    Validator      sdk.AccAddress `json:"validator"`
    Amount         sdk.Coins      `json:"amount"`
}

func RelayEvent(
  cdc *codec.Codec,
  cosmosRecipient sdk.AccAddress,
  validator sdk.AccAddress,
  nonce int,
  ethereumAddress string,
  amount sdk.Coins) error {

  fmt.Printf("\relayEvent() received:\n")
  fmt.Printf("\n Cosmos Recipient: %s,\n Validator: %s,\n Nonce: %d,\n Ethereum Address: %s,\n Amount: %s\n\n",
              cosmosRecipient, validator, nonce, ethereumAddress, amount) 


  cliCtx := context.NewCLIContext().
                WithCodec(cdc).
                WithAccountDecoder(cdc)

  txBldr := authtxb.NewTxBuilderFromCLI().
                WithTxEncoder(utils.GetTxEncoder(cdc))
  
  err := cliCtx.EnsureAccountExists();
  if err != nil {
    return err
  }

  // const bech32Validator, err2 = sdk.AccAddressFromBech32(activeValidator)
  // if err2 != nil {
  //     fmt.Errorf("%s", err2)
  // }

  //Get cosmos recipient
  // const cosmosRecipient, err1 = sdk.AccAddressFromBech32(lockEvent.CosmosRecipient.Hex())
  // if err1 != nil {
  //     fmt.Errorf("%s", err1)
  // }

  // TODO: Add token address to lockEvent struct for token parsing,
  //       if field == '0x00000000....' then use string 'ethereum'
  //       for decoding
  
  //Get coin amount and correct for wei 10**18
  // amount, err3 = sdk.ParseCoins(strings,Join(strconv.Itoa(lockEvent.Value/(Pow(10.0, 18))), "ethereum")
  // if err3 != nil {
  //     fmt.Errorf("%s", err3)
  // }

  ethBridgeClaim := types.NewEthBridgeClaim(nonce, ethereumAddress, cosmosRecipient, validator, amount)
  msg := types.NewMsgMakeEthBridgeClaim(ethBridgeClaim)
  
  err1 := msg.ValidateBasic()
  if err1 != nil {
    return err1
  }

  return utils.CompleteAndBroadcastTxCLI(txBldr, cliCtx, []sdk.Msg{msg})

}
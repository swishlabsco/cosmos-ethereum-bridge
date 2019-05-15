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

func relayEvent(
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

  ethereumEvent := EthEvent(nonce, ethereumAddress, cosmosRecipient, validator, amount)
  msg := types.NewMsgMakeEthBridgeClaim(ethereumEvent)
  
  err = msg.ValidateBasic()
  if err != nil {
    return err
  }

  //TODO: build and sign the transaction first
  //
  //  msg := utils.buildUnsignedStdTx(txBldr, cliCtx, []sdk.Msg{msg})
  //  txBytes := utils.SignStdTxWithSignerAddress(msg)
  //
  //TODO: broadcast to a Tendermint node
  //
  //  res, err := cliCtx.BroadcastTx(txBytes)
  //  if err != nil {
  //      return err
  //  }
  //  return cliCtx.PrintOutput(res)

}
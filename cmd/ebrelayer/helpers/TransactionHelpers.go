package helpers

import (
	"github.com/cosmos/cosmos-sdk/client/context"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/x/auth"
  authcmd "github.com/cosmos/cosmos-sdk/x/auth/client/cli"
	clientrest "github.com/cosmos/cosmos-sdk/client/rest"

	tmcrypto "github.com/tendermint/tendermint/crypto"
)

// clientrest.WriteGenerateStdTxResponse(w, cdc, cliCtx, baseReq, []sdk.Msg{msg})

// Build, sign and broadcast a message with a keybase accountName and password
func BuildSignAndBroadcastMsg(cdc *codec.Codec, cliCtx context.CLIContext, txCtx authcmd.stdTx, accountName string,
	password string, msg sdk.Msg) ([]byte, error) {
	// build and sign
	txBytes, err := txCtx.BuildAndSign(accountName, password, []sdk.Msg{msg})
	if err != nil {
		return nil, err
	}

	// broadcast
	res, err := cliCtx.BroadcastTxAsync(txBytes)
	if err != nil {
		return nil, err
	}

	output, err := codec.MarshalJSONIndent(cdc, res)
	if err != nil {
		return nil, err
	}
	return output, err
}

// Build, sign and broadcast a message with a private key, not using account from keybase
func PrivBuildSignAndBroadcastMsg(cdc *codec.Codec, cliCtx context.CLIContext, txCtx authcmd.stdTx,
	priv tmcrypto.PrivKey, msg sdk.Msg) ([]byte, error) {
	// build
	stdSignMsg, err := txCtx.Build([]sdk.Msg{msg})
	if err != nil {
		return nil, err
	}

	//sign
	txBytes, err := privSign(txCtx, priv, stdSignMsg)
	if err != nil {
		return nil, err
	}

	// send
	res, err := cliCtx.BroadcastTxAsync(txBytes)
	if err != nil {
		return nil, err
	}

	output, err := codec.MarshalJSONIndent(cdc, res)
	if err != nil {
		return nil, err
	}
	return output, err
}

//Sign a transaction with a given private key
func privSign(txCtx authcmd.stdTx, priv tmcrypto.PrivKey, msg auth.StdSignMsg) ([]byte, error) {
	sig, err := priv.Sign(msg.Bytes())
	if err != nil {
		return nil, err
	}
	pubkey := priv.PubKey()

	sigs := []auth.StdSignature{{
		AccountNumber: msg.AccountNumber,
		Sequence:      msg.Sequence,
		PubKey:        pubkey,
		Signature:     sig,
	}}

	return txCtx.Codec.MarshalBinary(auth.NewStdTx(msg.Msgs, msg.Fee, sigs, msg.Memo))
}

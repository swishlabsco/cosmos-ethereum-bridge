package server

import (
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"net/http"
	"os"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/keys"
	clienttx "github.com/cosmos/cosmos-sdk/client/tx"
	clientrest "github.com/cosmos/cosmos-sdk/client/rest"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/rest"
	"github.com/cosmos/cosmos-sdk/version"
	"github.com/cosmos/cosmos-sdk/x/auth"
	"github.com/cosmos/cosmos-sdk/x/bank"
	"github.com/cosmos/cosmos-sdk/codec"

	app "github.com/swishlabsco/cosmos-ethereum-bridge"
	"github.com/swishlabsco/cosmos-ethereum-bridge/x/ethbridge"
)

const (
	name1 = "validtor1"
	name2 = "validator2"
	name3 = "validator99"
	ethereumAddress = "0x7B95B6EC7EbD73572298cEf32Bb54FA408207359"
	cosmosRecipient = "cosmos1gn8409qq9hnrxde37kuxwx5hrxpfpv8426szuv"
	testValidator = "cosmos1xdp5tvt7lxh8rf9xx07wy2xlagzhq24ha48xtq"
	amount = 1500
	nonce = 5
	pw    = app.DefaultKeyPass
	pw2 = "12345678"
)

var fees = sdk.Coins{sdk.NewInt64Coin(sdk.DefaultBondDenom, 5)}

func init() {
	mintkey.BcryptSecurityParameter = 1
}

// Test that the server starts correctly
func TestConnectionStatus(t *testing.T) {
	cleanup, _, _, port := InitTestServer(t, 1, []sdk.AccAddress{}, true)
	defer cleanup()
	getNodeInfo(t, port)
	getSyncStatus(t, port, false)
}

// Check that we can query blocks
func TestBlock(t *testing.T) {
	cleanup, _, _, port := InitTestServer(t, 1, []sdk.AccAddress{}, true)
	defer cleanup()
	getBlock(t, port, -100, false)
	getBlock(t, port, 100000000, true)
}

func TestValidators(t *testing.T) {
	cleanup, _, _, port := InitTestServer(t, 1, []sdk.AccAddress{}, true)
	defer cleanup()
	resultVals := getValidatorSets(t, port, -1, false)

	// Check that the validator's address and public key is in the result set
	require.Contains(t, resultVals.Validators[0].Address.String(), "cosmosvalcons")
	require.Contains(t, resultVals.Validators[0].PubKey, "cosmosvalconspub")

	getValidatorSets(t, port, 5, false)
	getValidatorSets(t, port, 10000000, true)
}

// Test encoding of transactions from txs/encode
func TestEncodeTx(t *testing.T) {
	//Start the server
	kb, err := keys.NewKeyBaseFromDir(InitClientHome(t, ""))
	require.NoError(t, err)
	addr, seed := CreateAddr(t, name1, pw, kb)
	cleanup, _, _, port := InitTestServer(t, 1, []sdk.AccAddress{addr}, true)
	defer cleanup()

	// Test transfer
	res, body, _ := doTransferWithGas(t, port, seed, name1, memo, "", addr, "2", 1, false, false, fees)
	var tx auth.StdTx
	cdc.UnmarshalJSON([]byte(body), &tx)

	req := clienttx.EncodeReq{Tx: tx}
	encodedJSON, _ := cdc.MarshalJSON(req)
	res, body = Request(t, port, "POST", "/txs/encode", encodedJSON)

	// Make response is valid and able to be decoded
	require.Equal(t, http.StatusOK, res.StatusCode, body)
	encodeResp := struct {
		Tx string `json:"tx"`
	}{}

	require.Nil(t, cdc.UnmarshalJSON([]byte(body), &encodeResp))

	// Verifiy that the base64 can be decoded
	decodedBytes, err := base64.StdEncoding.DecodeString(encodeResp.Tx)
	require.Nil(t, err)

	// Check that the transaction decodes
	var decodedTx auth.StdTx
	require.Nil(t, cdc.UnmarshalBinaryLengthPrefixed(decodedBytes, &decodedTx))
	require.Equal(t, memo, decodedTx.Memo)
}

// Test signing and broadcasting a simple coin transfer transaction
func TestTxSignAndBroadcast(t *testing.T) {
	// Start the server...
	kb, err := keys.NewKeyBaseFromDir(InitClientHome(t, ""))
	require.NoError(t, err)
	addr, seed := CreateAddr(t, name1, pw, kb)
	cleanup, _, _, port := InitTestServer(t, 1, []sdk.AccAddress{addr}, true)

	defer cleanup()
	acc := getAccount(t, port, addr)

	// Simulate basic transaction and check the result code status
	res, body, _ := doTransferWithGas(
		t, port, seed, name1, memo, "", addr, client.GasFlagAuto, 1.0, true, false, fees,
	)
	require.Equal(t, http.StatusOK, res.StatusCode, body)

	// Check the estimated gas cost
	var gasEstResp rest.GasEstimateResponse
	require.Nil(t, cdc.UnmarshalJSON([]byte(body), &gasEstResp))
	require.NotZero(t, gasEstResp.GasEstimate)

	// Generate the transaction
	gas := fmt.Sprintf("%d", gasEstResp.GasEstimate)
	res, body, _ = doTransferWithGas(t, port, seed, name1, memo, "", addr, gas, 1, false, false, fees)
	require.Equal(t, http.StatusOK, res.StatusCode, body)

	// Test each attribute of the transaction
	var tx auth.StdTx
	require.Nil(t, cdc.UnmarshalJSON([]byte(body), &tx))
	require.Equal(t, len(tx.Msgs), 1)
	require.Equal(t, tx.Msgs[0].Route(), "bank")
	require.Equal(t, tx.Msgs[0].GetSigners(), []sdk.AccAddress{addr})
	require.Equal(t, 0, len(tx.Signatures))
	require.Equal(t, memo, tx.Memo)
	require.NotZero(t, tx.Fee.Gas)

	// Sign and broadcast the transaction
	gasEstimate := int64(tx.Fee.Gas)
	_, body = signAndBroadcastGenTx(t, port, name1, pw, body, acc, 1.0, false)

	// Check if tx was committed
	var txResp sdk.TxResponse
	require.Nil(t, cdc.UnmarshalJSON([]byte(body), &txResp))
	require.Equal(t, uint32(0), txResp.Code)
	require.Equal(t, gasEstimate, txResp.GasWanted)
}

// Test make bridge claim
func TestMakeBridgeClaim(t *testing.T) {
	// Initalize the test server
	kb, err := keys.NewKeyBaseFromDir(InitClientHome(t, ""))
	require.NoError(t, err)
	addr, seed := CreateAddr(t, name1, pw, kb)
	cleanup, _, _, port := InitTestServer(t, 1, []sdk.AccAddress{addr}, true)
	defer cleanup()

	// Get an account and it's initial balance
	acc := getAccount(t, port, addr)
	initialBalance := acc.GetCoins()

	// Create MakeBridgeClaim TX
	ethBridgeClaim := types.NewEthBridgeClaim(nonce, ethereumSender, cosmosRecipient, validator, amount)
	msg := types.NewMsgMakeEthBridgeClaim(ethBridgeClaim)
	err = msg.ValidateBasic()
	if err != nil {
		return err
	}

	tests.WaitForHeight(resultTx.Height+1, port)

	// Check if tx was committed
	require.Equal(t, uint32(0), := .Code)

	// Query the commited TX
	txs = getTransactions(t, port, fmt.Sprintf("action=make%%20claim&validator=%s", addr.String()))
	require.Equal(t, emptyTxs, txs)

	var claimID uint64
	cdc.MustUnmarshalBinaryLengthPrefixed(:= .Data, &claimID)

	// Check that the claim was created with the correct values
	claim := getClaim(t, port, claimID)
	require.Equal(t, ethereumSender, claim.getEthereumSender())
	require.Equal(t, cosmosRecipient, claim.getCosmosRecipient())
	require.Equal(t, amount, claim.getAmount())
	require.Equal(t, nonce, claim.getNonce())
	require.Equal(t, testValidator, claim.getValidator())

	// Confirm that this is the account which submitted the claim
	validator := getValidator(t, port, claimID)
	require.Equal(t, addr.String(), claim.Validator) 
	require.Equal(t, claimId, claim.claimID)
}

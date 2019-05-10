package main

import (
  "bytes"
  "encoding/base64"
  "encoding/hex"
  "encoding/json"
  "fmt"
  "os"
  "strconv"
  "strings"

  sdk "github.com/cosmos/cosmos-sdk/types"
  "github.com/cosmos/cosmos-sdk/x/auth"

  "github.com/spf13/cobra"
  "github.com/tendermint/tendermint/crypto"
  "github.com/tendermint/tendermint/crypto/ed25519"

  app "github.com/swishlabsco/cosmos-ethereum-bridge"
  relayer "github.com/swishlabsco/cosmos-ethereum-bridge/cmd/ebrelayer/relayer"
)

func init() {

  config := sdk.GetConfig()
  config.SetBech32PrefixForAccount(sdk.Bech32PrefixAccAddr, sdk.Bech32PrefixAccPub)
  config.SetBech32PrefixForValidator(sdk.Bech32PrefixValAddr, sdk.Bech32PrefixValPub)
  config.SetBech32PrefixForConsensusNode(sdk.Bech32PrefixConsAddr, sdk.Bech32PrefixConsPub)
  config.Seal()

  rootCmd.AddCommand(txCmd)
  rootCmd.AddCommand(pubkeyCmd)
  rootCmd.AddCommand(addrCmd)
  rootCmd.AddCommand(rawBytesCmd)
}

var rootCmd = &cobra.Command{
  Use:          "ebrelayer",
  Short:        "ethereum bridge relayer",
  SilenceUsage: true,
}

var relayerCmd = &cobra.Command{
  Use:   "relayer",
  Short: "Start a light client daemon which listens for ethereum txs",
  RunE:  runRelayerCmd,
}

var txCmd = &cobra.Command{
  Use:   "tx",
  Short: "Decode a bridge tx from hex or base64",
  RunE:  runTxCmd,
}

var pubkeyCmd = &cobra.Command{
  Use:   "pubkey",
  Short: "Decode a pubkey from hex, base64, or bech32",
  RunE:  runPubKeyCmd,
}

var addrCmd = &cobra.Command{
  Use:   "addr",
  Short: "Convert an address between hex and bech32",
  RunE:  runAddrCmd,
}

var rawBytesCmd = &cobra.Command{
  Use:   "raw-bytes",
  Short: "Convert raw bytes output (eg. [88 121 19 30]) to hex",
  RunE:  runRawBytesCmd,
}

func runRelayerCmd(cmd *cobra.Command, args []string) error {
  if len(args) != 6 {
    return fmt.Errorf("Expected string arguments:",
                      "chainId",
                      "validatorPassword",
                      "peggyAddress",
                      "eventSignature",
                      "validatorPrefix",
                      "validatorPassword")
  }

  // Parse chain's ID
  chainID := args[0]
  if chainID == "" {
    return fmt.Errorf("Must specify chain id")
  }

  // Parse ethereum provider (infura)
  ethereumProvider := args[1]
  if ethereumProvider == "" {
    return fmt.Errorf("Must specify ethreum network provider")
  }

  // Parse peggy's deployed contract address
  peggyContractAddress := args[2]
  if peggyContractAddress == "" {
    return fmt.Errorf("Must specify peggy contract address")
  }

  // Parse the event signature for the subscription
  eventSignature := args[3]
  if eventSignature == "" {
    return fmt.Errorf("Must specify event signature for subscription")
  }

  // Parse validator prefix
  validatorPrefix := args[4]
  if validatorPrefix == "" {
    return fmt.Errorf("Must specify validator's prefix")
  }

  // Parse validator password
  validatorPassword := args[5]
  if validatorPassword == "" {   //TODO: Sanitize input
    return fmt.Errorf("Must specify validator's password")
  }

  // // Initialize the relayer
  // err := relayer.InitRelayer([
  //   chainID,
  //   ethereumProvider,
  //   peggyContractAddress,
  //   eventSignature,
  //   validatorPrefix,
  //   validatorPassword
  // ])

  // if err != nil {
  //   return fmt.Errorf(err)
  // }

  return fmt.Errorf("Relayer timed out")
}

func runRawBytesCmd(cmd *cobra.Command, args []string) error {
  if len(args) != 1 {
    return fmt.Errorf("Expected single arg")
  }
  stringBytes := args[0]
  stringBytes = strings.Trim(stringBytes, "[")
  stringBytes = strings.Trim(stringBytes, "]")
  spl := strings.Split(stringBytes, " ")

  byteArray := []byte{}
  for _, s := range spl {
    b, err := strconv.Atoi(s)
    if err != nil {
      return err
    }
    byteArray = append(byteArray, byte(b))
  }
  fmt.Printf("%X\n", byteArray)
  return nil
}

func runPubKeyCmd(cmd *cobra.Command, args []string) error {
  if len(args) != 1 {
    return fmt.Errorf("Expected single arg")
  }

  pubkeyString := args[0]
  var pubKeyI crypto.PubKey

  // try hex, then base64, then bech32
  pubkeyBytes, err := hex.DecodeString(pubkeyString)
  if err != nil {
    var err2 error
    pubkeyBytes, err2 = base64.StdEncoding.DecodeString(pubkeyString)
    if err2 != nil {
      var err3 error
      pubKeyI, err3 = sdk.GetAccPubKeyBech32(pubkeyString)
      if err3 != nil {
        var err4 error
        pubKeyI, err4 = sdk.GetValPubKeyBech32(pubkeyString)

        if err4 != nil {
          var err5 error
          pubKeyI, err5 = sdk.GetConsPubKeyBech32(pubkeyString)
          if err5 != nil {
            return fmt.Errorf(`Expected hex, base64, or bech32. Got errors:
                hex: %v,
                base64: %v
                bech32 Acc: %v
                bech32 Val: %v
                bech32 Cons: %v`,
              err, err2, err3, err4, err5)
          }

        }
      }

    }
  }

  var pubKey ed25519.PubKeyEd25519
  if pubKeyI == nil {
    copy(pubKey[:], pubkeyBytes)
  } else {
    pubKey = pubKeyI.(ed25519.PubKeyEd25519)
    pubkeyBytes = pubKey[:]
  }

  cdc := app.MakeCodec()
  pubKeyJSONBytes, err := cdc.MarshalJSON(pubKey)
  if err != nil {
    return err
  }
  accPub, err := sdk.Bech32ifyAccPub(pubKey)
  if err != nil {
    return err
  }
  valPub, err := sdk.Bech32ifyValPub(pubKey)
  if err != nil {
    return err
  }

  consenusPub, err := sdk.Bech32ifyConsPub(pubKey)
  if err != nil {
    return err
  }
  fmt.Println("Address:", pubKey.Address())
  fmt.Printf("Hex: %X\n", pubkeyBytes)
  fmt.Println("JSON (base64):", string(pubKeyJSONBytes))
  fmt.Println("Bech32 Acc:", accPub)
  fmt.Println("Bech32 Validator Operator:", valPub)
  fmt.Println("Bech32 Validator Consensus:", consenusPub)
  return nil
}

func runAddrCmd(cmd *cobra.Command, args []string) error {
  if len(args) != 1 {
    return fmt.Errorf("Expected single arg")
  }

  addrString := args[0]
  var addr []byte

  // try hex, then bech32
  var err error
  addr, err = hex.DecodeString(addrString)
  if err != nil {
    var err2 error
    addr, err2 = sdk.AccAddressFromBech32(addrString)
    if err2 != nil {
      var err3 error
      addr, err3 = sdk.ValAddressFromBech32(addrString)

      if err3 != nil {
        return fmt.Errorf(`Expected hex or bech32. Got errors:
      hex: %v,
      bech32 acc: %v
      bech32 val: %v
      `, err, err2, err3)

      }
    }
  }

  accAddr := sdk.AccAddress(addr)
  valAddr := sdk.ValAddress(addr)

  fmt.Println("Address:", addr)
  fmt.Printf("Address (hex): %X\n", addr)
  fmt.Printf("Bech32 Acc: %s\n", accAddr)
  fmt.Printf("Bech32 Val: %s\n", valAddr)
  return nil
}

func runTxCmd(cmd *cobra.Command, args []string) error {
  if len(args) != 1 {
    return fmt.Errorf("Expected single arg")
  }

  txString := args[0]

  // try hex, then base64
  txBytes, err := hex.DecodeString(txString)
  if err != nil {
    var err2 error
    txBytes, err2 = base64.StdEncoding.DecodeString(txString)
    if err2 != nil {
      return fmt.Errorf(`Expected hex or base64. Got errors:
      hex: %v,
      base64: %v
      `, err, err2)
    }
  }

  var tx = auth.StdTx{}
  cdc := app.MakeCodec()

  // TODO: Update so it works with 26 bytes, not 31.
  err = cdc.UnmarshalBinaryLengthPrefixed(txBytes, &tx)
  if err != nil {
    return err
  }

  bz, err := cdc.MarshalJSON(tx)
  if err != nil {
    return err
  }

  buf := bytes.NewBuffer([]byte{})
  err = json.Indent(buf, bz, "", "  ")
  if err != nil {
    return err
  }

  fmt.Println(buf.String())
  return nil
}

func main() {
  err := rootCmd.Execute()
  if err != nil {
    os.Exit(1)
  }
  os.Exit(0)
}
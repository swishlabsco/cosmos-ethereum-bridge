package main

import (
  // "bytes"
  "encoding/base64"
  "encoding/hex"
  // "encoding/json"
  "fmt"
  "os"
  "strconv"
  "strings"

  "github.com/cosmos/cosmos-sdk/client"
  "github.com/cosmos/cosmos-sdk/client/keys"
  "github.com/cosmos/cosmos-sdk/client/rpc"
  "github.com/cosmos/cosmos-sdk/client/lcd"
  amino "github.com/tendermint/go-amino"
  "github.com/tendermint/tendermint/libs/cli"


  sdk "github.com/cosmos/cosmos-sdk/types"
  authcmd "github.com/cosmos/cosmos-sdk/x/auth/client/cli"
  auth "github.com/cosmos/cosmos-sdk/x/auth/client/rest"
  bankcmd "github.com/cosmos/cosmos-sdk/x/bank/client/cli"
  bank "github.com/cosmos/cosmos-sdk/x/bank/client/rest"

  "github.com/spf13/cobra"
  "github.com/tendermint/tendermint/crypto"
  "github.com/tendermint/tendermint/crypto/ed25519"

  app "github.com/swishlabsco/cosmos-ethereum-bridge"
  relayer "github.com/swishlabsco/cosmos-ethereum-bridge/cmd/ebrelayer/relayer"
  txs "github.com/swishlabsco/cosmos-ethereum-bridge/cmd/ebrelayer/txs"

  ethbridgecmd "github.com/swishlabsco/cosmos-ethereum-bridge/x/ethbridge/client"
  ethbridge "github.com/swishlabsco/cosmos-ethereum-bridge/x/ethbridge/client/rest"

)

const (
  storeAcc       = "acc"
  routeEthbridge = "ethbridge"
)

var defaultCLIHome = os.ExpandEnv("$HOME/.ebrelayer")

func init() {

  cdc := app.MakeCodec()

  config := sdk.GetConfig()
  config.SetBech32PrefixForAccount(sdk.Bech32PrefixAccAddr, sdk.Bech32PrefixAccPub)
  config.SetBech32PrefixForValidator(sdk.Bech32PrefixValAddr, sdk.Bech32PrefixValPub)
  config.SetBech32PrefixForConsensusNode(sdk.Bech32PrefixConsAddr, sdk.Bech32PrefixConsPub)
  config.Seal()

  mc := []sdk.ModuleClients{
    ethbridgecmd.NewModuleClient(routeEthbridge, cdc),
  }

  // Construct Root Command
  rootCmd.AddCommand(
    rpc.StatusCommand(),
    initRelayerCmd(cdc, mc),
    client.LineBreak,
    lcd.ServeCommand(cdc, registerRoutes),
    client.LineBreak,
    keys.Commands(),
    client.LineBreak,
  )

  rootCmd.AddCommand(pubkeyCmd)
  rootCmd.AddCommand(addrCmd)
  rootCmd.AddCommand(rawBytesCmd)

  executor := cli.PrepareMainCmd(rootCmd, "EBRELAYER", defaultCLIHome)
  err := executor.Execute()
  if err != nil {
    panic(err)
  }
}

var rootCmd = &cobra.Command{
  Use:          "ebrelayer",
  Short:        "ethereum bridge relayer",
  SilenceUsage: true,
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

func registerRoutes(rs *lcd.RestServer) {
 rs.CliCtx = rs.CliCtx.WithAccountDecoder(rs.Cdc)
 rpc.RegisterRoutes(rs.CliCtx, rs.Mux)
 txs.RegisterRoutes(rs.CliCtx, rs.Mux, rs.Cdc)
 auth.RegisterRoutes(rs.CliCtx, rs.Mux, rs.Cdc, storeAcc)
 bank.RegisterRoutes(rs.CliCtx, rs.Mux, rs.Cdc, rs.KeyBase)
 ethbridge.RegisterRoutes(rs.CliCtx, rs.Mux, rs.Cdc, routeEthbridge)
}

func initRelayerCmd(cdc *amino.Codec, mc []sdk.ModuleClients) *cobra.Command {
  initRelayerCmd := &cobra.Command{
    Use:     "init",
    Short:   "initalize relayer service",
    RunE:  RunRelayerCmd,
  }

  initRelayerCmd.AddCommand(
    //TODO: bankcmd/authcmd can likely be dropped
    bankcmd.SendTxCmd(cdc),
    client.LineBreak,
    authcmd.GetSignCommand(cdc),

    //TODO: add SendEventCommand here
    txs.GetBroadcastCommand(cdc),
    client.LineBreak,
  )

  for _, m := range mc {
    initRelayerCmd.AddCommand(m.GetTxCmd())
  }

  return initRelayerCmd
}

// -------------------------------------------------------------------------
// Initalizes the relayer service
// -------------------------------------------------------------------------
// Testing parameters:
//    validator = sdk.AccAddress("cosmos1xdp5tvt7lxh8rf9xx07wy2xlagzhq24ha48xtq")
//    chainId = "testing"
//    ethereumProvider = "wss://ropsten.infura.io/ws"
//    peggyContractAddress = "0xe56143b75f4eeac5fa80dc6ffd912d4a3ed21fdf"
//    eventSignature = "LogLock(address,address,uint256)"

func RunRelayerCmd(cmd *cobra.Command, args []string) error {
  // Parse chain's ID
  chainId := args[0]
  if chainId == "" {
    return fmt.Errorf("Must specify chain id")
  }

  // Parse ethereum provider
  ethereumProvider := args[1]
  if ethereumProvider == "wss://ropsten.infura.io/ws" {
    return fmt.Errorf("Only the ropsten ethereum network is currently supported")
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

  // TODO: Authenticate validator by their credentials instead
  //       of passing it as a parameter (see functions below)
  // Parse the validator running the relayer service
  validator := sdk.AccAddress(args[4])
  if validator == nil {
    return fmt.Errorf("Must have a validator for operations")
  }

  err := relayer.InitRelayer(
    // TODO: add codec back into this file
    // cdc,
    chainId,
    ethereumProvider,
    peggyContractAddress,
    eventSignature,
    validator)

  if err != nil {
    fmt.Printf("Relayer service closed.")
  }

  return nil
}


// TODO: use these to authenticate validator before launching relayer
//
// // Files containing the validator's unique cerification key
// func validateCertKeyFiles(certFile, keyFile string) error {
//   if keyFile == "" {
//     return errors.New("a key file is required")
//   }
//   if _, err := os.Stat(certFile); err != nil {
//     return err
//   }
//   if _, err := os.Stat(keyFile); err != nil {
//     return err
//   }
//   return nil
// }

// // Decode validator's certification key from file
// func readCertKeyFile(certFile string) (string, error) {
//   f, err := os.Open(certFile)
//   if err != nil {
//     return "", err
//   }
//   defer f.Close()
//   data, err := ioutil.ReadAll(f)
//   if err != nil {
//     return "", err
//   }
//   block, _ := pem.Decode(data)
//   if block == nil {
//     return "", fmt.Errorf("couldn't find required data in %s", certFile)
//   }
//   return validatorCertKey(block.Bytes)
// }


// // Validator's unique certification key
// func validatorCertKey(certBytes []byte) (string, error) {
//   cert, err := x509.ParseCertificate(certBytes)
//   if err != nil {
//     return "", err
//   }
//   h := sha256.New()
//   h.Write(cert.Raw)
//   certKeyBytes := h.Sum(nil)
//   var buf bytes.Buffer
//   for i, b := range certKeyBytes {
//     if i > 0 {
//       fmt.Fprintf(&buf, ":")
//     }
//     fmt.Fprintf(&buf, "%02X", b)
//   }
//   return fmt.Sprintf("Hashed certification key:%s", buf.String()), nil
// }

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

func main() {
  err := rootCmd.Execute()
  if err != nil {
    os.Exit(1)
  }
  os.Exit(0)
}
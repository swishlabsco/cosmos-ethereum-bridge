package main

import (
  "bytes"
  "encoding/base64"
  "encoding/hex"
  "encoding/json"
  "fmt"
  "os"

  "github.com/spf13/cobra"
  app "github.com/swishlabsco/cosmos-ethereum-bridge"
  authcmd "github.com/cosmos/cosmos-sdk/x/auth/client/cli"
  auth "github.com/cosmos/cosmos-sdk/x/auth/client/rest"
  // "github.com/swishlabsco/cosmos-ethereum-bridge/cmd/ebrelayer/listener"
  // "github.com/swishlabsco/cosmos-ethereum-bridge/cmd/ebrelayer/account"
  "github.com/swishlabsco/cosmos-ethereum-bridge/cmd/ebrelayer/txs"
  oracleclient "github.com/swishlabsco/cosmos-ethereum-bridge/x/oracle/client"
  oraclerest "github.com/swishlabsco/cosmos-ethereum-bridge/x/oracle/client/rest"
)

const (
  storeOracle = "oracle"
)

// DefaultRelayerHome sets the folder where the applcation data and configuration will be stored
// var DefaultRelayerHome = os.ExpandEnv("$HOME/.ebrelayer")

func init() {
  rootCmd.AddCommand(txCmd)
  // rootCmd.AddCommand(initCmd)
  // rootCmd.AddCommand(txCmd)
  // rootCmd.AddCommand(txServerCmd)
  // rootCmd.AddCommand(validateCmd)
}

var rootCmd = &cobra.Command{
  Use:          "ebrelayer",
  Short:        "Relayer tool",
}

// var initCmd := &cobra.Command{
//   Use:   "init",
//   Short: "Initialize the relayer"
//   RunE:  txs.initRelayer(cdc),
// }

var txCmd = &cobra.Command{
  Use:   "tx",
  Short: "Transaction subcommands for creating transactions between spam accounts",
  Rune: runTxCmd,
}

// var txServerCmd = &cobra.Command{
//   Use:   "tx-decoding-server",
//   Short: "Starts a server that listens to a unix socket to decode an oracle tx from hex or base64",
//   RunE:  runTxServerCmd,
// }

// var validateCmd = &cobra.Command{
//   Use:   "auto-validate",
//   Short: "Starts a server that listens to a unix socket to decode an oracle tx from hex or base64",
//   RunE:  runTxServerCmd,
// }

func main() {
  err := rootCmd.Execute()
  if err != nil {
    os.Exit(1)
  }
  os.Exit(0)
}

func runTxCmd(cdc *amino.Codec, mc []sdk.ModuleClients) *cobra.Command {

  cdc := app.MakeCodec()

  txCmd.AddCommand(
    bankcmd.SendTxCmd(cdc),
    client.LineBreak,
    authcmd.GetSignCommand(cdc),
    tx.GetBroadcastCommand(cdc),
    client.LineBreak,
  )

  mc := []sdk.ModuleClients{
    oracleclient.NewModuleClient(storeOracle, cdc),
  }

  for _, m := range mc {
    txCmd.AddCommand(m.GetTxCmd())
  }

  return txCmd
}

// func runTxCmd(_ *cobra.Command, args []string) error {
//   if len(args) != 1 {
//     return fmt.Errorf("Expected single arg")
//   }

//   txString := args[0]

//   // try hex, then base64
//   txBytes, err := hex.DecodeString(txString)
//   if err != nil {
//     var err2 error
//     txBytes, err2 = base64.StdEncoding.DecodeString(txString)
//     if err2 != nil {
//       return fmt.Errorf(`Expected hex or base64. Got errors:
//       hex: %v,
//       base64: %v
//       `, err, err2)
//     }
//   }


// func runTxServerCmd(_ *cobra.Command, _ []string) error {
//   os.Remove("/tmp/thorchaindebug-tx-decoding.sock")
//   l, err := net.Listen("unix", "/tmp/thorchaindebug-tx-decoding.sock")

//   if err != nil {
//     return fmt.Errorf("listen error %v", err)
//   }

//   defer l.Close()

//   cdc := thorchain.MakeCodec()

//   for {
//     c, err := l.Accept()
//     if err != nil {
//       return fmt.Errorf("accept error %v", err)
//     }

//     go txServer(c, cdc)
//   }
// }

// func txServer(c net.Conn, cdc *codec.Codec) {
//   for {
//     buf := make([]byte, 10240)
//     nr, err := c.Read(buf)
//     if err != nil {
//       if err == io.EOF {
//         // connection closed => just return the server
//         // fmt.Println("Connection closed")
//         return
//       }
//       fmt.Println("Could not read:", err)
//       continue
//     }

//     txString := string(buf[0:nr])

//     // fmt.Println("Server got tx:", txString)

//     // try hex, then base64
//     txBytes, err := hex.DecodeString(txString)
//     if err != nil {
//       var err2 error
//       txBytes, err2 = base64.StdEncoding.DecodeString(txString)
//       if err2 != nil {
//         fmt.Printf(`Expected hex or base64. Got errors:
//         hex: %v,
//         base64: %v
//         `, err, err2)
//         continue
//       }
//     }

//     var tx = auth.StdTx{}

//     err = cdc.UnmarshalBinary(txBytes, &tx)
//     if err != nil {
//       fmt.Println("Unmarshal binary error:", err)
//       continue
//     }

//     bz, err := cdc.MarshalJSON(tx)
//     if err != nil {
//       fmt.Println("Marshal json error:", err)
//       continue
//     }

//     buff := bytes.NewBuffer([]byte{})
//     err = json.Indent(buff, bz, "", "  ")
//     if err != nil {
//       fmt.Println("Json indent error:", err)
//       continue
//     }

//     // fmt.Println(buff.String())
//     // continue

//     _, err = c.Write(buff.Bytes())
//     if err != nil {
//       fmt.Println("Write error:", err)
//     }
//   }
// }

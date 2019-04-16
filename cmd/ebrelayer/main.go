package main

import (
  "github.com/spf13/cobra"
  "github.com/swishlabsco/cosmos-ethereum-bridge/app"
  "github.com/swishlabsco/cosmos-ethereum-bridge/cmd/ebrelayer/listener"
  "github.com/swishlabsco/cosmos-ethereum-bridge/cmd/ebrelayer/account"
  "github.com/swishlabsco/cosmos-ethereum-bridge/cmd/ebrelayer/txs"

  "github.com/tendermint/tendermint/libs/cli"
)

// -------------------------------------------------------------------------
// Starts an ethereum contract listener and tx relayer from the given
// cli arguments and flags
// -------------------------------------------------------------------------
func main() {

  cdc := app.MakeCodec()
  cobra.EnableCommandSorting = false

  rootCmd := &cobra.Command{
    Use:   "ebrelayer",
    Short: "Relay ethereum transactions to the bridge oracle",
  }

  // --- listener commands ---

  listenCmd := &cobra.Command{
    Use:   "listener",
    Short: "Starts an infura web socket which listens for ethereum transactions",
    RunE:  listener.start(),
  }

  listenCmd.Flags().String(listener.FlagNetwork, "", "The ethereum network to be monitored")
  listenCmd.Flags().String(listener.FlagContract, "", "The deployed contract's address")
  listenCmd.Flags().String(listener.FlagEventSig, "", "The event name to filter for")

  rootCmd.AddCommand(listenCmd)

  // --- tx relay commands ---

  txCmd := &cobra.Command{
    Use:   "tx",
    Short: "Relay subcommands for relaying transactions",
  }

  relayCmd := &cobra.Command{
    Use:   "relay",
    Short: "Relays an observed transactions to the Oracle",
    RunE:  txs.GetRelayCmd(cdc),
  }

  relayCmd.Flags().String(txs.FlagValidatorPrefix, "validator", "Prefix for the name of validator account keys")
  relayCmd.Flags().String(txs.FlagValidatorPassword, "", "Password for validator account keys")
  relayCmd.Flags().String(txs.FlagChainID, "", "Chain ID of tendermint node")
  relayCmd.Flags().String(txs.FlagNode, "tcp://localhost:26657", "<host>:<port> to tendermint rpc interface for this chain")

  txsCmd.AddCommand(relayCmd)

  rootCmd.AddCommand(txsCmd)

  // prepare and add flags
  executor := cli.PrepareMainCmd(rootCmd, "GA", app.DefaultCLIHome)
  err := executor.Execute()
  if err != nil {
    panic(err)
  }
}
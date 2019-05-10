package server

import (
  "errors"
  "fmt"
  "net"
  "net/http"
  "os"

  "github.com/gorilla/mux"
  "github.com/rakyll/statik/fs"
  "github.com/spf13/cobra"
  "github.com/spf13/viper"
  "github.com/tendermint/tendermint/libs/log"
  "github.com/cosmos/cosmos-sdk/client/lcd/statik"
  rpcserver "github.com/tendermint/tendermint/rpc/lib/server"

  "github.com/cosmos/cosmos-sdk/client"
  "github.com/cosmos/cosmos-sdk/client/context"
  "github.com/cosmos/cosmos-sdk/codec"
  keybase "github.com/cosmos/cosmos-sdk/crypto/keys"
  "github.com/cosmos/cosmos-sdk/server"


)

// RestServer represents the Light Client Rest server
type Server struct {
  Mux     *mux.Router
  CliCtx  context.CLIContext
  KeyBase keybase.Keybase
  Cdc     *codec.Codec

  log           log.Logger
  listener      net.Listener
  ceritifcaion  string
}

// NewServer creates a new rest server instance
func NewServer(cdc *codec.Codec) *Server {
  r := mux.NewRouter()
  cliCtx := context.NewCLIContext().WithCodec(cdc).WithAccountDecoder(cdc)

  return &Server{
    Mux:    r,
    CliCtx: cliCtx,
    Cdc:    cdc,

    log: logger,
  }
}

// Start starts the rest server
func (rs *Server) InitServer(listenAddr string, sslHosts string,
  certFile string, keyFile string, maxOpen int, secure bool) (err error) {

  server.TrapSignal(func() {
    err := rs.listener.Close()
    rs.log.Error("error closing listener", "err", err)
  })

  rs.listener, err = rpcserver.Listen(
    listenAddr,
    rpcserver.Config{MaxOpenConnections: maxOpen},
  )
  if err != nil {
    return
  }
  rs.log.Info(fmt.Sprintf("Starting Bridge server (id: %q)...",
    viper.GetString(client.FlagChainID)))

  if !secure {
    return rpcserver.StartHTTPServer(rs.listener, rs.Mux, rs.log)
  }

  // handle certificates
  if certFile != "" {
    if err := validateCertKeyFiles(certFile, keyFile); err != nil {
      return err
    }

    //  cert/key pair is provided, read the cerification key file
    rs.certification, err = readCertKeyFile(certFile)
    if err != nil {
      return err
    }
  }


    defer func() {
      os.Remove(certFile)
      os.Remove(keyFile)
    }()
  }

  rs.log.Info(rs.certification)
  return rpcserver.StartHTTPAndTLSServer(
    rs.listener,
    rs.Mux,
    certFile, keyFile,
    rs.log,
  )
}

// ServeCommand will start an ebd light client daemon
func ServeCommand(cdc *codec.Codec, registerRoutes func(*Server)) *cobra.Command {
  cmd := &cobra.Command{
    Use:   "server",
    Short: "Start server for local light client",
    RunE: func(cmd *cobra.Command, args []string) (err error) {
      rs := NewServer(cdc)

      registerRoutesFn(rs)

      // Start the server
      err = rs.Start(
        viper.GetString(client.FlagListenAddr),
        viper.GetString(client.FlagSSLHosts),
        viper.GetString(client.FlagSSLCertFile),
        viper.GetString(client.FlagSSLKeyFile),
        viper.GetInt(client.FlagMaxOpenConnections),
        viper.GetBool(client.FlagTLS))

      return err
    },
  }

  return client.RegisterRestServerFlags(cmd)
}

// Statik for server
func (rs *Server) registerSwaggerUI() {
  statikFS, err := fs.New()
  if err != nil {
    panic(err)
  }
  staticServer := http.FileServer(statikFS)
  rs.Mux.PathPrefix("/swagger-ui/").Handler(http.StripPrefix("/swagger-ui/", staticServer))
}

// Files containing the validator's unique cerification key
func validateCertKeyFiles(certFile, keyFile string) error {
  if keyFile == "" {
    return errors.New("a key file is required")
  }
  if _, err := os.Stat(certFile); err != nil {
    return err
  }
  if _, err := os.Stat(keyFile); err != nil {
    return err
  }
  return nil
}

// Decode validator's certification key from file
func readCertKeyFile(certFile string) (string, error) {
  f, err := os.Open(certFile)
  if err != nil {
    return "", err
  }
  defer f.Close()
  data, err := ioutil.ReadAll(f)
  if err != nil {
    return "", err
  }
  block, _ := pem.Decode(data)
  if block == nil {
    return "", fmt.Errorf("couldn't find required data in %s", certFile)
  }
  return validatorCertKey(block.Bytes)
}


// Validator's unique certification key
func validatorCertKey(certBytes []byte) (string, error) {
  cert, err := x509.ParseCertificate(certBytes)
  if err != nil {
    return "", err
  }
  h := sha256.New()
  h.Write(cert.Raw)
  certKeyBytes := h.Sum(nil)
  var buf bytes.Buffer
  for i, b := range certKeyBytes {
    if i > 0 {
      fmt.Fprintf(&buf, ":")
    }
    fmt.Fprintf(&buf, "%02X", b)
  }
  return fmt.Sprintf("Hashed certification key:%s", buf.String()), nil
}

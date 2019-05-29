package txs

import (
  "log"

  "github.com/swishlabsco/cosmos-ethereum-bridge/cmd/ebrelayer/events"
  // "github.com/swishlabsco/cosmos-ethereum-bridge/x/ethbridge/types"
)

type WitnessClaim struct {
  Nonce          int            `json:"nonce"`
  EthereumSender string         `json:"ethereum_sender"`
  CosmosReceiver sdk.AccAddress `json:"cosmos_receiver"`
  Validator      sdk.AccAddress `json:"validator"`
  Amount         sdk.Coins      `json:"amount"`
}

func ParsePayloadAndRelay(cdc *codec.Codec, validator sdk.accAddress, eventPayload *EventPayload) (err) {
  // Set the witnessClaim's validator
  var witnessClaim WitnessClaim
  witnessClaim.Validator = validator

  // Get the keyset of the payload's fields
  payloadKeySet := events.Keys(eventPayload);

  // Parse each key field individually
  for field := range payloadKeySet {
      switch(field):
          case "_id":
              // Print the unique id of the event.
              // TODO: Replace the 'eventHash' in events with this unique id (?)
              fmt.Print(field);
          case "_from":
              ethereumSender, ok := field.Address();
              if !ok {
                  return eventPayload, errors.New("Error while parsing transaction's ethereum sender");
              }
              witnessClaim.EthereumSender = ethereumSender;
          case "_to":
              cosmosRecipient, ok := field.Bytes32();

              // TODO: Convert this to Cosmos address type using 'sdk.AccAddressFromBech32' (?)
              // -------------------------------------------------------------------------
              // const bech32Validator, err2 = sdk.AccAddressFromBech32(activeValidator)
              // if err2 != nil {
              //     fmt.Errorf("%s", err2)
              // }
              // -------------------------------------------------------------------------

              if !ok {
                  return eventPayload, errors.New("Error while parsing transaction's Cosmos recipient");
              }
              witnessClaim.CosmosRecipient = cosmosRecipient;
          case "_token":
              tokenType, ok := field.Bytes32();
              if !ok {
                  return eventPayload, errors.New("Error while parsing the token type");
              }
              witnessClaim.Token = tokenType;               
          case "_value":
              amount, ok := field.BigInt()
              if !ok {
                  return eventPayload, errors.New("Error while parsing transaction's value")
              }
              // Correct for wei 10**18
              //    TODO: Once TokenType is implemented on ethbridgemessage, make sure this
              //          wei conversion correctly handles erc20 tokens.
              weiAmount, err = sdk.ParseCoins(strings,Join(strconv.Itoa(amount/(Pow(10.0, 18))), "ethereum")
              if err3 != nil {
                  fmt.Errorf("%s", err3)
              }
              witnessClaim.Amount = weiAmount
          case "_nonce":
              nonce, ok := field.BigInt()
              if !ok {
                  return eventPayload, errors.New("Error while parsing transaction's nonce")
              }
              witnessClaim.Nonce = nonce
  }

  err := relay(cdc,
        witnessClaim.CosmosRecipient,
        //witnessClaim.Token,
          // TODO: Token type doesn't exist on ethbridgeclaim,
          //       it must be added before can include it in the relay.
        witnessClaim.Validator
        witnessClaim.Nonce,
        witnessClaim.EthereumSender,
        witnessClaim.Amount)

  if err != nil {
    log.Fatal(err)
  }
}

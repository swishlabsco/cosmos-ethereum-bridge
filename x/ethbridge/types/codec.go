package types

import (
	"github.com/cosmos/cosmos-sdk/codec"
)

// RegisterCodec registers concrete types on the Amino codec
func RegisterCodec(cdc *codec.Codec) {
	cdc.RegisterConcrete(MsgMakeEthBridgeClaim{}, "oracle/MsgMakeEthBridgeClaim", nil)
}

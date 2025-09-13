package types

import (
	"github.com/cosmos/cosmos-sdk/codec"
	cdctypes "github.com/cosmos/cosmos-sdk/codec/types"
)

// NOTE: No interfaces to register yet (no Msg/Query). Keep for future.
func RegisterInterfaces(reg cdctypes.InterfaceRegistry) {}

var (
	// Amino is required by some SDK internals; keep minimal.
	Amino = codec.NewLegacyAmino()
	// ModuleCdc is used for JSON (genesis) where needed.
	ModuleCdc = codec.NewProtoCodec(cdctypes.NewInterfaceRegistry())
)

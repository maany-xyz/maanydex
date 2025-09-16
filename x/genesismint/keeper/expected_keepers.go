package keeper

import (
    "context"

    sdk "github.com/cosmos/cosmos-sdk/types"
    icacontrollertypes "github.com/cosmos/ibc-go/v8/modules/apps/27-interchain-accounts/controller/types"
    exported "github.com/cosmos/ibc-go/v8/modules/core/exported"
    connectiontypes "github.com/cosmos/ibc-go/v8/modules/core/03-connection/types"
)

type BankKeeper interface {
  MintCoins(ctx context.Context, moduleName string, amt sdk.Coins) error
  SendCoinsFromModuleToAccount(ctx context.Context, moduleName string, recipient sdk.AccAddress, amt sdk.Coins) error
}


type ClientKeeper interface {
  GetClientConsensusState(ctx sdk.Context, clientID string, height exported.Height) (exported.ConsensusState, bool)
}

// ICA controller + channel keepers needed for Go-only ICA flow.
// Minimal subset mirroring x/interchaintxs/types/expected_keepers.go
// to avoid taking a dependency on that module here.

type ICAControllerKeeper interface {
    GetActiveChannelID(ctx sdk.Context, connectionID, portID string) (string, bool)
    GetInterchainAccountAddress(ctx sdk.Context, connectionID, portID string) (string, bool)
    GetParams(ctx sdk.Context) icacontrollertypes.Params
}

type ICAControllerMsgServer interface {
    RegisterInterchainAccount(context.Context, *icacontrollertypes.MsgRegisterInterchainAccount) (*icacontrollertypes.MsgRegisterInterchainAccountResponse, error)
    SendTx(context.Context, *icacontrollertypes.MsgSendTx) (*icacontrollertypes.MsgSendTxResponse, error)
}

type ConnectionKeeper interface {
    GetConnection(ctx sdk.Context, connectionID string) (connectiontypes.ConnectionEnd, bool)
}

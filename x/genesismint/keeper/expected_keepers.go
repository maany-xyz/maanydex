package keeper

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	clienttypes "github.com/cosmos/ibc-go/v8/modules/core/02-client/types"
	exported "github.com/cosmos/ibc-go/v8/modules/core/exported"
)

type BankKeeper interface {
  MintCoins(ctx sdk.Context, moduleName string, amt sdk.Coins) error
  SendCoinsFromModuleToAccount(ctx sdk.Context, moduleName string, recipient sdk.AccAddress, amt sdk.Coins) error
}

type ClientKeeper interface {
	GetClientConsensusState(ctx sdk.Context, clientID string, height clienttypes.Height) (exported.ConsensusState, bool)
}
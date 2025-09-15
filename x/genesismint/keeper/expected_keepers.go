package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
	exported "github.com/cosmos/ibc-go/v8/modules/core/exported"
)

type BankKeeper interface {
  MintCoins(ctx context.Context, moduleName string, amt sdk.Coins) error
  SendCoinsFromModuleToAccount(ctx context.Context, moduleName string, recipient sdk.AccAddress, amt sdk.Coins) error
}


type ClientKeeper interface {
  GetClientConsensusState(ctx sdk.Context, clientID string, height exported.Height) (exported.ConsensusState, bool)
}
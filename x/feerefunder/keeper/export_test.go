package keeper

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/maany-xyz/maany-dex/v5/x/feerefunder/types"
)

func (k Keeper) CheckFees(ctx sdk.Context, fees types.Fee) error {
	return k.checkFees(ctx, fees)
}

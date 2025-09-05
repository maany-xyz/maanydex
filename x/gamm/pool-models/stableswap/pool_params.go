package stableswap

import (
	"github.com/maany-xyz/maany-dex/v5/osmomath"
	"github.com/maany-xyz/maany-dex/v5/x/gamm/types"
)

func (params PoolParams) Validate() error {
	if params.ExitFee.IsNegative() {
		return types.ErrNegativeExitFee
	}

	if params.ExitFee.GTE(osmomath.OneDec()) {
		return types.ErrTooMuchExitFee
	}

	if params.SwapFee.IsNegative() {
		return types.ErrNegativeSpreadFactor
	}

	if params.SwapFee.GTE(osmomath.OneDec()) {
		return types.ErrTooMuchSpreadFactor
	}
	return nil
}

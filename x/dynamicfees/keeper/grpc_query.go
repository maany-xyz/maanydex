package keeper

import (
	"github.com/maany-xyz/maany-dex/v5/x/dynamicfees/types"
)

var _ types.QueryServer = Keeper{}

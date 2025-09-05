package keeper

import (
	"github.com/maany-xyz/maany-dex/v5/x/interchaintxs/types"
)

var _ types.QueryServer = Keeper{}

package keeper

import (
	"github.com/maany-xyz/maany-dex/v5/x/cron/types"
)

var _ types.QueryServer = Keeper{}

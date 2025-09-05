package keeper_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	testkeeper "github.com/maany-xyz/maany-dex/v5/testutil/interchaintxs/keeper"
	"github.com/maany-xyz/maany-dex/v5/x/interchaintxs/types"
)

func TestGetParams(t *testing.T) {
	k, ctx := testkeeper.InterchainTxsKeeper(t, nil, nil, nil, nil, nil, nil, nil)
	params := types.DefaultParams()

	err := k.SetParams(ctx, params)
	require.NoError(t, err)

	require.EqualValues(t, params, k.GetParams(ctx))
}

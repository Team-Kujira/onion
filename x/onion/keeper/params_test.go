package keeper_test

import (
	"testing"

	"github.com/stretchr/testify/require"

    keepertest "onion/testutil/keeper"
    "onion/x/onion/types"
)

func TestGetParams(t *testing.T) {
	k, ctx := keepertest.OnionKeeper(t)
	params := types.DefaultParams()

	require.NoError(t, k.SetParams(ctx, params))
	require.EqualValues(t, params, k.GetParams(ctx))
}

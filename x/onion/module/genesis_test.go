package onion_test

import (
	"testing"

	keepertest "onion/testutil/keeper"
	"onion/testutil/nullify"
	onion "onion/x/onion/module"
	"onion/x/onion/types"
	"github.com/stretchr/testify/require"
)

func TestGenesis(t *testing.T) {
	genesisState := types.GenesisState{
		Params:	types.DefaultParams(),
		
		// this line is used by starport scaffolding # genesis/test/state
	}

	k, ctx := keepertest.OnionKeeper(t)
	onion.InitGenesis(ctx, k, genesisState)
	got := onion.ExportGenesis(ctx, k)
	require.NotNil(t, got)

	nullify.Fill(&genesisState)
	nullify.Fill(got)

	

	// this line is used by starport scaffolding # genesis/test/assert
}

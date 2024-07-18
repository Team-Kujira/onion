package keeper

import (
	"onion/x/onion/types"
)

var _ types.QueryServer = Keeper{}

package keeper

import (
	"fmt"

	"onion/x/onion/types"

	"github.com/cosmos/cosmos-sdk/baseapp"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"cosmossdk.io/core/store"
	"cosmossdk.io/log"
	txsigning "cosmossdk.io/x/tx/signing"
	"github.com/cosmos/cosmos-sdk/codec"
)

type (
	Keeper struct {
		cdc          codec.BinaryCodec
		storeService store.KVStoreService
		logger       log.Logger

		// the address capable of executing a MsgUpdateParams message. Typically, this
		// should be the x/gov module account.
		authority string

		accountKeeper   types.AccountKeeper
		router          *baseapp.MsgServiceRouter
		SignModeHandler *txsigning.HandlerMap
	}
)

// NewKeeper returns a new instance of the x/ibchooks keeper
func NewKeeper(
	cdc codec.BinaryCodec,
	storeService store.KVStoreService,
	logger log.Logger,
	authority string,
	router *baseapp.MsgServiceRouter,
	accountKeeper types.AccountKeeper,
	signModeHandler *txsigning.HandlerMap,
) *Keeper {
	if _, err := sdk.AccAddressFromBech32(authority); err != nil {
		panic(fmt.Sprintf("invalid authority address: %s", authority))
	}
	return &Keeper{
		cdc:             cdc,
		storeService:    storeService,
		authority:       authority,
		logger:          logger,
		router:          router,
		accountKeeper:   accountKeeper,
		SignModeHandler: signModeHandler,
	}
}

// GetAuthority returns the module's authority.
func (k Keeper) GetAuthority() string {
	return k.authority
}

// Logger returns a module-specific logger.
func (k Keeper) Logger() log.Logger {
	return k.logger.With("module", fmt.Sprintf("x/%s", types.ModuleName))
}

func (k Keeper) InitGenesis(ctx sdk.Context, genState types.GenesisState) {
	k.SetParams(ctx, genState.Params)
	for _, seq := range genState.Sequences {
		if err := k.SetSequence(ctx, seq); err != nil {
			panic(err)
		}
	}
}

func (k Keeper) ExportGenesis(ctx sdk.Context) *types.GenesisState {
	return &types.GenesisState{
		Params:    k.GetParams(ctx),
		Sequences: k.GetAllSequences(ctx),
	}
}

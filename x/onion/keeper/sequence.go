package keeper

import (
	"cosmossdk.io/store/prefix"
	storetypes "cosmossdk.io/store/types"
	"github.com/cosmos/cosmos-sdk/runtime"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/gogoproto/proto"

	"onion/x/onion/types"
)

// GetAuthorityMetadata returns the authority metadata for a specific denom
func (k Keeper) GetSequence(ctx sdk.Context, address string) (types.OnionSequence, error) {
	store := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	prefixStore := prefix.NewStore(store, []byte(types.OnionSequencePrefix))
	bz := prefixStore.Get([]byte(address))
	if bz == nil {
		return types.OnionSequence{
			Address:  address,
			Sequence: 0,
		}, nil
	}
	sequence := types.OnionSequence{}
	err := proto.Unmarshal(bz, &sequence)
	if err != nil {
		return types.OnionSequence{}, err
	}
	return sequence, nil
}

// SetAuthorityMetadata stores authority metadata for a specific denom
func (k Keeper) SetSequence(ctx sdk.Context, sequence types.OnionSequence) error {
	store := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	prefixStore := prefix.NewStore(store, []byte(types.OnionSequencePrefix))

	bz, err := proto.Marshal(&sequence)
	if err != nil {
		return err
	}

	prefixStore.Set([]byte(sequence.Address), bz)
	return nil
}

func (k Keeper) GetAllSequences(ctx sdk.Context) []types.OnionSequence {
	store := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	iterator := storetypes.KVStorePrefixIterator(store, []byte(types.OnionSequencePrefix))
	defer iterator.Close()

	sequences := []types.OnionSequence{}
	for ; iterator.Valid(); iterator.Next() {
		sequence := types.OnionSequence{}
		err := proto.Unmarshal(iterator.Value(), &sequence)
		if err != nil {
			panic(err)
		}
		sequences = append(sequences, sequence)
	}
	return sequences
}

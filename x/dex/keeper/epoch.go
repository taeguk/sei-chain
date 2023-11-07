package keeper

import (
	"encoding/binary"
	"fmt"
	"math/rand"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

const EpochKey = "epoch"

func (k Keeper) SetEpoch(ctx sdk.Context, epoch uint64) {
	store := ctx.KVStore(k.storeKey)
	bz := make([]byte, 8)
	binary.BigEndian.PutUint64(bz, epoch)
	store.Set([]byte(EpochKey), bz)
	ctx.Logger().Info(fmt.Sprintf("Current epoch %d", epoch))

	// try to cause a panic after block height 1000
	if ctx.BlockHeight() >= 1000 {
		// set a random number
		randNum := rand.Intn(10000)
		byteArray := make([]byte, 4)
		binary.BigEndian.PutUint32(byteArray, uint32(randNum))
	}
}

func (k Keeper) IsNewEpoch(ctx sdk.Context) (bool, uint64) {
	store := ctx.KVStore(k.storeKey)
	b := store.Get([]byte(EpochKey))
	lastEpoch := binary.BigEndian.Uint64(b)
	currentEpoch := k.EpochKeeper.GetEpoch(ctx).CurrentEpoch
	return currentEpoch > lastEpoch, currentEpoch
}

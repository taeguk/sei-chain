package keeper

import (
	"encoding/json"

	sdk "github.com/cosmos/cosmos-sdk/types"
	ethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/sei-protocol/sei-chain/x/evm/types"
)

type Logs struct {
	Ls []*ethtypes.Log `json:"logs"`
}

func (k *Keeper) AddLog(ctx sdk.Context, l *ethtypes.Log) error {
	// TODO: potentially decorate log with block/tx metadata
	store := k.PrefixStore(ctx, types.TransientModuleStateKeyPrefix)
	logs := Logs{Ls: []*ethtypes.Log{}}
	ls, err := k.GetLogs(ctx)
	if err != nil {
		return err
	}
	logs.Ls = append(ls, l)
	logsbz, err := json.Marshal(&logs)
	if err != nil {
		return err
	}
	store.Set(types.LogsKey, logsbz)
	return nil
}

func (k *Keeper) GetLogs(ctx sdk.Context) ([]*ethtypes.Log, error) {
	store := k.PrefixStore(ctx, types.TransientModuleStateKeyPrefix)
	logsbz := store.Get(types.LogsKey)
	logs := Logs{Ls: []*ethtypes.Log{}}
	if logsbz == nil {
		return []*ethtypes.Log{}, nil
	}
	if err := json.Unmarshal(logsbz, &logs); err != nil {
		return []*ethtypes.Log{}, err
	}
	return logs.Ls, nil
}

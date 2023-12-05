package evm

import (
	"math/big"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/common"
	"github.com/sei-protocol/sei-chain/x/evm/keeper"
	"github.com/sei-protocol/sei-chain/x/evm/state"
	"github.com/sei-protocol/sei-chain/x/evm/types"
)

func InitGenesis(ctx sdk.Context, k *keeper.Keeper, genState types.GenesisState) {
	k.InitGenesis(ctx)
	k.SetParams(ctx, genState.Params)
	s := state.NewDBImpl(ctx, k, false)
	for _, entry := range genState.Alloc {
		balance := new(big.Int).SetBytes(entry.Account.Balance)
		address := common.HexToAddress(entry.Address)
		s.SetBalance(address, balance)
	}
}

func ExportGenesis(ctx sdk.Context, k *keeper.Keeper) *types.GenesisState {
	genesis := types.DefaultGenesis()
	genesis.Params = k.GetParams(ctx)

	return genesis
}

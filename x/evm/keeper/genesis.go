package keeper

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	"github.com/ethereum/go-ethereum/common"

	"github.com/sei-protocol/sei-chain/x/evm/types"
)

const FeeCollectorAddress = "41cc5d9842746c69d689c8379f7f5662b8701393"

func (k Keeper) InitGenesis(ctx sdk.Context) {
	moduleAcc := authtypes.NewEmptyModuleAccount(types.ModuleName, authtypes.Minter, authtypes.Burner)
	k.accountKeeper.SetModuleAccount(ctx, moduleAcc)

	k.SetParams(ctx, types.DefaultParams())

	// set FeeCollectorName association with a randomly generated ethereum address hash
	evmAddrFc := common.HexToAddress(FeeCollectorAddress)
	seiAddrFc := k.accountKeeper.GetModuleAddress(authtypes.FeeCollectorName)
	k.SetAddressMapping(ctx, seiAddrFc, evmAddrFc)
}

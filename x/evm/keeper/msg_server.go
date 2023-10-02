package keeper

import (
	"context"
	"errors"
	"fmt"
	"math"
	"math/big"

	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	ethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/sei-protocol/sei-chain/x/evm/state"
	"github.com/sei-protocol/sei-chain/x/evm/types"
	tmtypes "github.com/tendermint/tendermint/types"
)

type msgServer struct {
	Keeper
}

// NewMsgServerImpl returns an implementation of the MsgServer interface
// for the provided Keeper.
func NewMsgServerImpl(keeper Keeper) types.MsgServer {
	return &msgServer{Keeper: keeper}
}

var _ types.MsgServer = msgServer{}

func (server msgServer) EVMTransaction(goCtx context.Context, msg *types.MsgEVMTransaction) (*types.MsgEVMTransactionResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	stateDB := state.NewStateDBImpl(ctx, &server)
	tx, _ := msg.AsTransaction()
	cfg := server.GetChainConfig(ctx).EthereumConfig(server.ChainID())

	var gp core.GasPool
	if ctx.BlockGasMeter().Limit() == 0 {
		// infinite gas meter
		gp = math.MaxUint64
	} else {
		gp = core.GasPool(ctx.BlockGasMeter().Limit() - ctx.BlockGasMeter().GasConsumedToLimit())
	}
	baseFee := server.GetBaseFee()
	coinbase, err := server.GetFeeCollectorAddress(ctx)
	if err != nil {
		return &types.MsgEVMTransactionResponse{}, err
	}
	blockCtx := vm.BlockContext{
		CanTransfer: core.CanTransfer,
		Transfer:    core.Transfer,
		GetHash:     server.GetHashFn(ctx),
		Coinbase:    coinbase,
		GasLimit:    gp.Gas(),
		BlockNumber: big.NewInt(ctx.BlockHeight()),
		Time:        uint64(ctx.BlockHeader().Time.Unix()),
		Difficulty:  big.NewInt(0), // only needed for PoW
		BaseFee:     baseFee,       // feemarket not enabled
		Random:      nil,           // not supported
	}
	signer := ethtypes.MakeSigner(cfg, big.NewInt(ctx.BlockHeight()), uint64(ctx.BlockTime().Unix()))
	emsg, err := core.TransactionToMessage(tx, signer, baseFee)
	if err != nil {
		return &types.MsgEVMTransactionResponse{}, err
	}
	txCtx := core.NewEVMTxContext(emsg)
	evmInstance := vm.NewEVM(blockCtx, txCtx, stateDB, cfg, vm.Config{})

	st := core.NewStateTransition(evmInstance, emsg, &gp)
	res, err := st.TransitionDb()
	if err != nil {
		return &types.MsgEVMTransactionResponse{}, err
	}
	// write to DB, among other things
	if err := stateDB.Finalize(); err != nil {
		return &types.MsgEVMTransactionResponse{}, err
	}

	return &types.MsgEVMTransactionResponse{
		GasUsed:    res.UsedGas,
		ReturnData: res.ReturnData,
		VmError:    res.Err.Error(),
		Hash:       tx.Hash().Hex(),
	}, nil
}

// returns a function that provides block header hash based on block number
func (server msgServer) GetHashFn(ctx sdk.Context) vm.GetHashFunc {
	return func(height uint64) common.Hash {
		if height > math.MaxInt64 {
			ctx.Logger().Error("Sei block height is bounded by int64 range")
			return common.Hash{}
		}
		h := int64(height)
		if ctx.BlockHeight() == h {
			// current header hash is in the context already
			return common.BytesToHash(ctx.HeaderHash())
		}
		if ctx.BlockHeight() < h {
			// future block doesn't have a hash yet
			return common.Hash{}
		}
		// fetch historical hash from historical info
		histInfo, found := server.stakingKeeper.GetHistoricalInfo(ctx, h)
		if !found {
			// too old, already pruned
			return common.Hash{}
		}
		header, err := tmtypes.HeaderFromProto(&histInfo.Header)
		if err != nil {
			// parsing issue
			ctx.Logger().Error(fmt.Sprintf("failed to parse historical info header %s due to %s", histInfo.Header.String(), err))
			return common.Hash{}
		}

		return common.BytesToHash(header.Hash())
	}
}

// fee market is not enabled for now, so returning 0
func (server msgServer) GetBaseFee() *big.Int {
	return big.NewInt(0)
}

func (server msgServer) GetFeeCollectorAddress(ctx sdk.Context) (common.Address, error) {
	moduleAddr := server.accountKeeper.GetModuleAddress(authtypes.FeeCollectorName)
	if evmAddr, ok := server.GetEVMAddress(ctx, moduleAddr); !ok {
		return common.Address{}, errors.New("fee collector's EVM address not found")
	} else {
		return evmAddr, nil
	}
}

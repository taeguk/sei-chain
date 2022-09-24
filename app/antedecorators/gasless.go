package antedecorators

import (
	"bytes"

	sdk "github.com/cosmos/cosmos-sdk/types"
	dextypes "github.com/sei-protocol/sei-chain/x/dex/types"
	nitrokeeper "github.com/sei-protocol/sei-chain/x/nitro/keeper"
	nitrotypes "github.com/sei-protocol/sei-chain/x/nitro/types"
	oraclekeeper "github.com/sei-protocol/sei-chain/x/oracle/keeper"
	oracletypes "github.com/sei-protocol/sei-chain/x/oracle/types"
)

type GaslessDecorator struct {
	wrapped      []sdk.AnteDecorator
	oracleKeeper oraclekeeper.Keeper
	nitroKeeper  nitrokeeper.Keeper
}

func NewGaslessDecorator(wrapped []sdk.AnteDecorator, oracleKeeper oraclekeeper.Keeper, nitroKeeper nitrokeeper.Keeper) GaslessDecorator {
	return GaslessDecorator{wrapped: wrapped, oracleKeeper: oracleKeeper, nitroKeeper: nitroKeeper}
}

func (gd GaslessDecorator) AnteHandle(ctx sdk.Context, tx sdk.Tx, simulate bool, next sdk.AnteHandler) (newCtx sdk.Context, err error) {
	if !isTxGasless(tx, ctx, gd.oracleKeeper, gd.nitroKeeper) {
		// if not gasless, then we use the wrappers

		// AnteHandle always takes a `next` so we need a no-op to execute only one handler at a time
		terminatorHandler := func(ctx sdk.Context, _ sdk.Tx, _ bool) (sdk.Context, error) {
			return ctx, nil
		}
		// iterating instead of recursing the handler for readability
		// we use blank here because we shouldn't handle the error
		for _, handler := range gd.wrapped {
			ctx, _ = handler.AnteHandle(ctx, tx, simulate, terminatorHandler)
		}
		return next(ctx, tx, simulate)
	}
	gaslessMeter := sdk.NewInfiniteGasMeter()

	return next(ctx.WithGasMeter(gaslessMeter), tx, simulate)
}

func isTxGasless(tx sdk.Tx, ctx sdk.Context, oracleKeeper oraclekeeper.Keeper, nitroKeeper nitrokeeper.Keeper) bool {
	if len(tx.GetMsgs()) == 0 {
		// empty TX shouldn't be gasless
		return false
	}
	for _, msg := range tx.GetMsgs() {
		switch m := msg.(type) {
		case *dextypes.MsgPlaceOrders:
			if DexPlaceOrdersIsGasless(m) {
				continue
			}
			return false
		case *dextypes.MsgCancelOrders:
			if DexCancelOrdersIsGasless(m) {
				continue
			}
			return false
		case *oracletypes.MsgAggregateExchangeRateVote:
			if OracleVoteIsGasless(m, ctx, oracleKeeper) {
				continue
			}
			return false
		case *nitrotypes.MsgRecordTransactionData:
			if NitroRecordTxDataGasless(m, ctx, nitroKeeper) {
				continue
			}
			return false
		default:
			return false
		}
	}
	return true
}

func DexPlaceOrdersIsGasless(msg *dextypes.MsgPlaceOrders) bool {
	return true
}

// WhitelistedGaslessCancellationAddrs TODO: migrate this into params state
var WhitelistedGaslessCancellationAddrs = []sdk.AccAddress{}

func DexCancelOrdersIsGasless(msg *dextypes.MsgCancelOrders) bool {
	return allSignersWhitelisted(msg)
}

func allSignersWhitelisted(msg *dextypes.MsgCancelOrders) bool {
	for _, signer := range msg.GetSigners() {
		isWhitelisted := false
		for _, whitelisted := range WhitelistedGaslessCancellationAddrs {
			if bytes.Compare(signer, whitelisted) == 0 { //nolint:gosimple
				isWhitelisted = true
				break
			}
		}
		if !isWhitelisted {
			return false
		}
	}
	return true
}

func OracleVoteIsGasless(msg *oracletypes.MsgAggregateExchangeRateVote, ctx sdk.Context, keeper oraclekeeper.Keeper) bool {
	feederAddr, err := sdk.AccAddressFromBech32(msg.Feeder)
	if err != nil {
		return false
	}

	valAddr, err := sdk.ValAddressFromBech32(msg.Validator)
	if err != nil {
		return false
	}

	err = keeper.ValidateFeeder(ctx, feederAddr, valAddr)
	if err != nil {
		return false
	}

	// this returns an error IFF there is no vote present
	// this also gets cleared out after every vote window, so if there is no vote present, we may want to allow gasless tx
	_, err = keeper.GetAggregateExchangeRateVote(ctx, valAddr)
	// if there is no error that means there is a vote present, so we dont allow gasless tx otherwise we allow it
	return err != nil
}

func NitroRecordTxDataGasless(msg *nitrotypes.MsgRecordTransactionData, ctx sdk.Context, keeper nitrokeeper.Keeper) bool {
	for _, signer := range msg.GetSigners() {
		if !keeper.IsTxSenderWhitelisted(ctx, signer.String()) {
			return false
		}
	}
	return true
}

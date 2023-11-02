package occ_tests

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	"testing"
	"time"
)

// TestParallelTransactions verifies that the store state is equivalent
// between both parallel and sequential executions
func TestParallelTransactions(t *testing.T) {
	tests := []struct {
		name    string
		runs    int
		shuffle bool
		txs     func(tCtx *TestContext) []sdk.Msg
	}{
		{
			name: "Test wasm instantiations",
			runs: 5,
			txs: func(tCtx *TestContext) []sdk.Msg {
				return joinMsgs(
					wasmInstantiate(tCtx, 10),
				)
			},
		},
		{
			name: "Test bank transfer",
			runs: 5,
			txs: func(tCtx *TestContext) []sdk.Msg {
				return joinMsgs(
					bankTransfer(tCtx, 10),
				)
			},
		},
		{
			name: "Test governance proposal",
			runs: 5,
			txs: func(tCtx *TestContext) []sdk.Msg {
				return joinMsgs(
					governanceSubmitProposal(tCtx, 10),
				)
			},
		},
		{
			name:    "Test combinations",
			runs:    5,
			shuffle: true,
			txs: func(tCtx *TestContext) []sdk.Msg {
				return joinMsgs(
					wasmInstantiate(tCtx, 10),
					bankTransfer(tCtx, 10),
					governanceSubmitProposal(tCtx, 10),
				)
			},
		},
	}

	for _, tt := range tests {
		blockTime := time.Now()
		signer := initSigner()

		// execute sequentially, then in parallel
		// the responses and state should match for both
		sCtx := initTestContext(signer, blockTime)
		txs := tt.txs(sCtx)
		if tt.shuffle {
			txs = shuffle(txs)
		}
		sResponse := runSequentially(sCtx, txs)

		for i := 0; i < tt.runs; i++ {
			pCtx := initTestContext(signer, blockTime)
			pResponse := runParallel(pCtx, txs)
			assertEqualResponses(t, sResponse, pResponse)
			assertEqualStores(t, sCtx.Ctx, pCtx.Ctx)
		}
	}
}

func TestSimple(t *testing.T) {
	tests := []struct {
		name string
		runs int
		txs  func(tCtx *TestContext) []sdk.Msg
	}{
		{
			name: "Test bank transfer",
			runs: 1,
			txs: func(tCtx *TestContext) []sdk.Msg {
				return joinMsgs(
					bankTransfer(tCtx, 1),
				)
			},
		},
	}

	for _, tt := range tests {
		blockTime := time.Now()
		signer := initSigner()

		// execute sequentially, then in parallel
		// the responses and state should match for both
		sCtx := initTestContext(signer, blockTime)
		txs := tt.txs(sCtx)
		sResponse := runSequentially(sCtx, txs)

		for i := 0; i < tt.runs; i++ {
			pCtx := initTestContext(signer, blockTime)
			pResponse := runParallel(pCtx, txs)
			assertEqualResponses(t, sResponse, pResponse)
			assertEqualStores(t, sCtx.Ctx, pCtx.Ctx)
		}
	}
}

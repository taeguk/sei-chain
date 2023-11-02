package occ_tests

import (
	"context"
	tx2 "github.com/cosmos/cosmos-sdk/client/tx"
	types3 "github.com/cosmos/cosmos-sdk/codec/types"
	types2 "github.com/cosmos/cosmos-sdk/crypto/types"
	"github.com/cosmos/cosmos-sdk/testutil/testdata"
	txtype "github.com/cosmos/cosmos-sdk/types/tx"
	"github.com/cosmos/cosmos-sdk/types/tx/signing"
	authsigning "github.com/cosmos/cosmos-sdk/x/auth/signing"
	"github.com/cosmos/cosmos-sdk/x/auth/tx"
	"github.com/sei-protocol/sei-chain/app"
	"math/rand"
	"os"
	"testing"
	"time"

	wasmkeeper "github.com/CosmWasm/wasmd/x/wasm/keeper"
	wasmxtypes "github.com/CosmWasm/wasmd/x/wasm/types"
	"github.com/cosmos/cosmos-sdk/store"
	sdk "github.com/cosmos/cosmos-sdk/types"
	keepertest "github.com/sei-protocol/sei-chain/testutil/keeper"
	dexcache "github.com/sei-protocol/sei-chain/x/dex/cache"
	dextypes "github.com/sei-protocol/sei-chain/x/dex/types"
	dexutils "github.com/sei-protocol/sei-chain/x/dex/utils"
	minttypes "github.com/sei-protocol/sei-chain/x/mint/types"

	"github.com/stretchr/testify/require"
	"github.com/tendermint/tendermint/abci/types"
	tmproto "github.com/tendermint/tendermint/proto/tendermint/types"
)

const INSTANTIATE = `{"whitelist": ["sei1h9yjz89tl0dl6zu65dpxcqnxfhq60wxx8s5kag"],
    "use_whitelist":false,"admin":"sei1h9yjz89tl0dl6zu65dpxcqnxfhq60wxx8s5kag",
	"limit_order_fee":{"decimal":"0.0001","negative":false},
	"market_order_fee":{"decimal":"0.0001","negative":false},
	"liquidation_order_fee":{"decimal":"0.0001","negative":false},
	"margin_ratio":{"decimal":"0.0625","negative":false},
	"max_leverage":{"decimal":"4","negative":false},
	"default_base":"USDC",
	"native_token":"USDC","denoms": ["SEI","ATOM","USDC","SOL","ETH","OSMO","AVAX","BTC"],
	"full_denom_mapping": [["usei","SEI","0.000001"],["uatom","ATOM","0.000001"],["uusdc","USDC","0.000001"]],
	"funding_payment_lookback":3600,"spot_market_contract":"sei1h9yjz89tl0dl6zu65dpxcqnxfhq60wxx8s5kag",
	"supported_collateral_denoms": ["USDC"],
	"supported_multicollateral_denoms": ["ATOM"],
	"oracle_denom_mapping": [["usei","SEI","1"],["uatom","ATOM","1"],["uusdc","USDC","1"],["ueth","ETH","1"]],
	"multicollateral_whitelist": ["sei1h9yjz89tl0dl6zu65dpxcqnxfhq60wxx8s5kag"],
	"multicollateral_whitelist_enable": true,
	"funding_payment_pairs": [["USDC","ETH"]],
	"default_margin_ratios":{
		"initial":"0.3",
		"partial":"0.25",
		"maintenance":"0.06"
	}}`

func toRequests(testCtx *TestContext, msgs []sdk.Msg) []types.RequestDeliverTx {
	var txs []types.RequestDeliverTx
	tc := app.MakeEncodingConfig().TxConfig

	priv := testCtx.Signer.PrivateKey
	acct := testCtx.TestApp.AccountKeeper.GetAccount(testCtx.Ctx, testCtx.Signer.Sender)

	for _, m := range msgs {
		a, err := types3.NewAnyWithValue(m)
		if err != nil {
			panic(err)
		}

		tBuilder := tx.WrapTx(&txtype.Tx{
			Body: &txtype.TxBody{
				Messages: []*types3.Any{a},
			},
			AuthInfo: &txtype.AuthInfo{
				Fee: &txtype.Fee{
					Amount:   funds(10000000000),
					GasLimit: 10000000000,
					Payer:    testCtx.Signer.Sender.String(),
					Granter:  testCtx.Signer.Sender.String(),
				},
			},
		})

		err = tBuilder.SetSignatures(signing.SignatureV2{
			PubKey: priv.PubKey(),
			Data: &signing.SingleSignatureData{
				SignMode:  tc.SignModeHandler().DefaultMode(),
				Signature: nil,
			},
			Sequence: acct.GetSequence(),
		})
		if err != nil {
			panic(err)
		}

		signerData := authsigning.SignerData{
			ChainID:       testCtx.Ctx.ChainID(),
			AccountNumber: 0,
			Sequence:      acct.GetSequence(),
		}

		sigV2, err := tx2.SignWithPrivKey(
			tc.SignModeHandler().DefaultMode(), signerData,
			tBuilder, priv, tc, acct.GetSequence())

		if err != nil {
			panic(err)
		}

		err = tBuilder.SetSignatures(sigV2)
		if err != nil {
			panic(err)
		}

		b, err := tc.TxEncoder()(tBuilder.GetTx())
		if err != nil {
			panic(err)
		}
		txs = append(txs, types.RequestDeliverTx{Tx: b})

		if err := acct.SetSequence(acct.GetSequence() + 1); err != nil {
			panic(err)
		}
	}
	return txs
}

func funds(amount int64) sdk.Coins {
	return sdk.NewCoins(sdk.NewCoin("usei", sdk.NewInt(amount)))
}

type TestContext struct {
	Ctx    sdk.Context
	CodeID uint64

	Signer         Signer
	TestAccount1   sdk.AccAddress
	TestAccount2   sdk.AccAddress
	ContractKeeper *wasmkeeper.PermissionedKeeper
	TestApp        *app.App
}

type Signer struct {
	Sender     sdk.AccAddress
	PrivateKey types2.PrivKey
	PublicKey  types2.PubKey
}

func initSigner() Signer {
	priv1, pubKey, sender := testdata.KeyTestPubAddr()
	return Signer{
		Sender:     sender,
		PrivateKey: priv1,
		PublicKey:  pubKey,
	}
}

// initTestContext initializes a new TestContext with a new app and a new contract
func initTestContext(signer Signer, blockTime time.Time) *TestContext {
	contractFile := "../integration_test/contracts/mars.wasm"
	testApp := keepertest.TestApp()
	ctx := testApp.BaseApp.NewContext(false, tmproto.Header{Time: time.Now()})
	ctx = ctx.WithChainID("chainId")
	ctx = ctx.WithContext(context.WithValue(ctx.Context(), dexutils.DexMemStateContextKey, dexcache.NewMemState(testApp.GetMemKey(dextypes.MemStoreKey))))
	ctx = ctx.WithBlockGasMeter(sdk.NewGasMeter(100000000))
	ctx = ctx.WithBlockHeader(tmproto.Header{Height: ctx.BlockHeader().Height, ChainID: ctx.BlockHeader().ChainID, Time: blockTime})
	testAccount, _ := sdk.AccAddressFromBech32("sei1h9yjz89tl0dl6zu65dpxcqnxfhq60wxx8s5kag")
	depositAccount, _ := sdk.AccAddressFromBech32("sei1yezq49upxhunjjhudql2fnj5dgvcwjj87pn2wx")
	amounts := sdk.NewCoins(sdk.NewCoin("usei", sdk.NewInt(1000000000000000)), sdk.NewCoin("uusdc", sdk.NewInt(1000000000000000)))
	bankkeeper := testApp.BankKeeper
	bankkeeper.MintCoins(ctx, minttypes.ModuleName, amounts)
	bankkeeper.SendCoinsFromModuleToAccount(ctx, minttypes.ModuleName, testAccount, amounts)
	bankkeeper.MintCoins(ctx, minttypes.ModuleName, amounts)
	bankkeeper.SendCoinsFromModuleToAccount(ctx, minttypes.ModuleName, depositAccount, amounts)
	bankkeeper.MintCoins(ctx, minttypes.ModuleName, amounts)
	bankkeeper.SendCoinsFromModuleToAccount(ctx, minttypes.ModuleName, signer.Sender, amounts)

	wasm, err := os.ReadFile(contractFile)
	if err != nil {
		panic(err)
	}
	wasmKeeper := testApp.WasmKeeper
	contractKeeper := wasmkeeper.NewDefaultPermissionKeeper(&wasmKeeper)
	var perm *wasmxtypes.AccessConfig
	codeID, err := contractKeeper.Create(ctx, testAccount, wasm, perm)
	if err != nil {
		panic(err)
	}

	return &TestContext{
		Ctx:            ctx,
		CodeID:         codeID,
		Signer:         signer,
		TestAccount1:   testAccount,
		TestAccount2:   depositAccount,
		ContractKeeper: contractKeeper,
		TestApp:        testApp,
	}
}

var ignoredStoreKeys = map[string]struct{}{
	"mem_capability": {},
	"epoch":          {},
	"deferredcache":  {},
}

func joinMsgs(msgsList ...[]sdk.Msg) []sdk.Msg {
	var result []sdk.Msg
	for _, msgs := range msgsList {
		result = append(result, msgs...)
	}
	return result
}

func shuffle(msgs []sdk.Msg) []sdk.Msg {
	var result []sdk.Msg
	for _, i := range rand.Perm(len(msgs)) {
		result = append(result, msgs[i])
	}
	return result
}

func assertEqualStores(t *testing.T, expectedCtx sdk.Context, actualCtx sdk.Context) {
	expectedStoreKeys := expectedCtx.MultiStore().StoreKeys()
	actualStoreKeys := actualCtx.MultiStore().StoreKeys()
	require.Equal(t, len(expectedStoreKeys), len(actualStoreKeys))

	// store keys are mapped by reference, so Name()==Name() comparison is needed
	for _, esk := range expectedStoreKeys {
		for _, ask := range actualStoreKeys {
			_, ignored := ignoredStoreKeys[esk.Name()]
			if !ignored && (esk.Name() == ask.Name()) {
				expected := expectedCtx.MultiStore().GetKVStore(esk)
				actual := actualCtx.MultiStore().GetKVStore(ask)
				compareStores(t, esk, expected, actual)
			}
		}
	}
}

// compareStores compares the expected and actual KVStores regarding keys and values
func compareStores(t *testing.T, storeKey sdk.StoreKey, expected store.KVStore, actual store.KVStore) {
	if _, ok := ignoredStoreKeys[storeKey.Name()]; ok {
		return
	}

	iexpected := expected.Iterator(nil, nil)
	defer iexpected.Close()

	iactual := actual.Iterator(nil, nil)
	defer iactual.Close()

	// Iterate over the expected store
	for ; iexpected.Valid(); iexpected.Next() {
		key := iexpected.Key()
		expectedValue := iexpected.Value()

		// Ensure the key exists in the actual store
		actualValue := actual.Get(key)
		require.NotNil(t, actualValue, "key not found in the %s store: %s", storeKey.Name(), string(key))

		// Compare the values for the current key
		require.Equal(t, string(expectedValue), string(actualValue), "%s value mismatch for key: %s", storeKey.Name(), string(key))

		// Move to the next key in the actual store for the upcoming iteration
		iactual.Next()
	}

	// Ensure there are no extra keys in the actual store
	require.False(t, iactual.Valid(), "Extra key found in the actual store: %s", storeKey.Name())
}

func runSequentially(testCtx *TestContext, msgs []sdk.Msg) []*sdk.DeliverTxResult {
	reqs := toRequests(testCtx, msgs)
	var res []*sdk.DeliverTxResult
	for idx, req := range reqs {
		ctx := testCtx.Ctx.WithTxIndex(idx)
		resp := testCtx.TestApp.DeliverTx(ctx, req)
		res = append(res, &sdk.DeliverTxResult{Response: resp})
	}
	return res
}

func runParallel(testCtx *TestContext, msgs []sdk.Msg) []*sdk.DeliverTxResult {
	reqs := toRequests(testCtx, msgs)

	var entries []*sdk.DeliverTxEntry
	for _, req := range reqs {
		entries = append(entries, &sdk.DeliverTxEntry{Request: req})
	}

	res := testCtx.TestApp.DeliverTxBatch(testCtx.Ctx, sdk.DeliverTxBatchRequest{TxEntries: entries})
	return res.Results
}

func assertEqualResponses(t *testing.T, expected []*sdk.DeliverTxResult, actual []*sdk.DeliverTxResult) {

	if len(expected) != len(actual) {
		t.Fatalf("expected %d responses, got %d", len(expected), len(actual))
	}

	for i, r := range expected {
		if r.Response.Code != actual[i].Response.Code {
			t.Fatalf("expected expected code %d, got %d", r.Response.Code, actual[i].Response.Code)
		}
		if 0 != actual[i].Response.Code {
			t.Fatalf("expected expected code %d, got %d", 0, actual[i].Response.Code)
		}
		if r.Response.Log != actual[i].Response.Log {
			t.Fatalf("expected expected log %s, got %s", r.Response.Log, actual[i].Response.Log)
		}
		if r.Response.Info != actual[i].Response.Info {
			t.Fatalf("expected expected info %s, got %s", r.Response.Info, actual[i].Response.Info)
		}
	}
}

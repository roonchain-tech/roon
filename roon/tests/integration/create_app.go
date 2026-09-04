package integration

import (
	"encoding/json"
	"os"

	"github.com/cosmos/cosmos-sdk/client/flags"

	dbm "github.com/cosmos/cosmos-db"
	ibctesting "github.com/cosmos/ibc-go/v10/testing"

	"github.com/cosmos/evm"
	"github.com/cosmos/evm/roon"
	srvflags "github.com/cosmos/evm/server/flags"
	"github.com/cosmos/evm/testutil/constants"
	feemarkettypes "github.com/cosmos/evm/x/feemarket/types"
	evmtypes "github.com/cosmos/evm/x/vm/types"

	"cosmossdk.io/log"

	"github.com/cosmos/cosmos-sdk/baseapp"
	simutils "github.com/cosmos/cosmos-sdk/testutil/sims"
	minttypes "github.com/cosmos/cosmos-sdk/x/mint/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
)

// CreateEvmd creates an evm app for regular integration tests (non-mempool)
// This version uses a noop mempool to avoid state issues during transaction processing
func CreateEvmd(chainID string, evmChainID uint64, customBaseAppOptions ...func(*baseapp.BaseApp)) evm.EvmApp {
	// A temporary home directory is created and used to prevent race conditions
	// related to home directory locks in chains that use the WASM module.
	defaultNodeHome, err := os.MkdirTemp("", "roon-temp-homedir")
	if err != nil {
		panic(err)
	}

	db := dbm.NewMemDB()
	logger := log.NewNopLogger()
	loadLatest := true
	appOptions := NewAppOptionsWithFlagHomeAndChainID(defaultNodeHome, evmChainID)

	baseAppOptions := append(customBaseAppOptions, baseapp.SetChainID(chainID))

	return roon.NewExampleApp(
		logger,
		db,
		nil,
		loadLatest,
		appOptions,
		baseAppOptions...,
	)
}

// SetupEvmd initializes a new roon app with default genesis state.
// It is used in IBC integration tests to create a new roon app instance.
func SetupEvmd() (ibctesting.TestingApp, map[string]json.RawMessage) {
	defaultNodeHome, err := os.MkdirTemp("", "roon-temp-homedir")
	if err != nil {
		panic(err)
	}

	app := roon.NewExampleApp(
		log.NewNopLogger(),
		dbm.NewMemDB(),
		nil,
		true,
		NewAppOptionsWithFlagHomeAndChainID(defaultNodeHome, constants.EighteenDecimalsChainID),
	)
	// disable base fee for testing
	genesisState := app.DefaultGenesis()
	fmGen := feemarkettypes.DefaultGenesisState()
	fmGen.Params.NoBaseFee = true
	genesisState[feemarkettypes.ModuleName] = app.AppCodec().MustMarshalJSON(fmGen)
	// The ibc test chains fund the sender accounts with the EVM extended denom
	// (see testutil/ibc/chain.go), so the staking bond denom and mint denom must
	// match it, otherwise token sends and delegations using the bond denom fail
	// with insufficient funds.
	stakingGen := stakingtypes.DefaultGenesisState()
	stakingGen.Params.BondDenom = evmtypes.DefaultEVMExtendedDenom
	genesisState[stakingtypes.ModuleName] = app.AppCodec().MustMarshalJSON(stakingGen)
	mintGen := minttypes.DefaultGenesisState()
	mintGen.Params.MintDenom = evmtypes.DefaultEVMExtendedDenom
	genesisState[minttypes.ModuleName] = app.AppCodec().MustMarshalJSON(mintGen)

	return app, genesisState
}

func NewAppOptionsWithFlagHomeAndChainID(home string, evmChainID uint64) simutils.AppOptionsMap {
	return simutils.AppOptionsMap{
		flags.FlagHome:      home,
		srvflags.EVMChainID: evmChainID,
	}
}

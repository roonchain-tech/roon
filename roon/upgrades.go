package roon

import (
	"context"

	"github.com/cosmos/evm/x/vm/types"

	storetypes "cosmossdk.io/store/types"
	upgradetypes "cosmossdk.io/x/upgrade/types"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/bech32"
	"github.com/cosmos/cosmos-sdk/types/module"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	vestingtypes "github.com/cosmos/cosmos-sdk/x/auth/vesting/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
)

// UpgradeName defines the on-chain upgrade name for the sample EVMD upgrade
// from v0.5.0 to v0.6.0.
//
// NOTE: This upgrade defines a reference implementation of what an upgrade
// could look like when an application is migrating from EVMD version
// v0.4.0 to v0.5.x
const UpgradeName = "v0.5.0-to-v0.6.0"

func (app EVMD) RegisterUpgradeHandlers() {
	app.UpgradeKeeper.SetUpgradeHandler(
		UpgradeName,
		func(ctx context.Context, _ upgradetypes.Plan, fromVM module.VersionMap) (module.VersionMap, error) {
			sdkCtx := sdk.UnwrapSDKContext(ctx)
			sdkCtx.Logger().Debug("this is a debug level message to test that verbose logging mode has properly been enabled during a chain upgrade")

			// Chains bootstrapped with the upstream cosmos/evm binaries stored
			// every bech32 address string with the "cosmos" prefixes. Rewrite
			// them with this chain's prefixes before any module keeper logic
			// resolves those addresses (e.g. the mint begin blocker).
			if err := app.migrateLegacyAddressPrefixes(ctx); err != nil {
				return nil, err
			}

			app.BankKeeper.SetDenomMetaData(ctx, banktypes.Metadata{
				Description: "Example description",
				DenomUnits: []*banktypes.DenomUnit{
					{
						Denom:    "atest",
						Exponent: 0,
						Aliases:  nil,
					},
					{
						Denom:    "test",
						Exponent: 18,
						Aliases:  nil,
					},
				},
				Base:    "atest",
				Display: "test",
				Name:    "Test Token",
				Symbol:  "TEST",
				URI:     "example_uri",
				URIHash: "example_uri_hash",
			})

			// (Required for NON-18 denom chains *only)
			// Update EVM params to add Extended denom options
			// Ensure that this corresponds to the EVM denom
			// (tyically the bond denom)
			evmParams := app.EVMKeeper.GetParams(sdkCtx)
			evmParams.ExtendedDenomOptions = &types.ExtendedDenomOptions{ExtendedDenom: "atest"}
			err := app.EVMKeeper.SetParams(sdkCtx, evmParams)
			if err != nil {
				return nil, err
			}
			// Initialize EvmCoinInfo in the module store. Chains bootstrapped before v0.5.0
			// binaries never stored this information (it lived only in process globals),
			// so migrating nodes would otherwise see an empty EvmCoinInfo on upgrade.
			if err := app.EVMKeeper.InitEvmCoinInfo(sdkCtx); err != nil {
				return nil, err
			}
			return app.ModuleManager.RunMigrations(ctx, app.Configurator(), fromVM)
		},
	)

	upgradeInfo, err := app.UpgradeKeeper.ReadUpgradeInfoFromDisk()
	if err != nil {
		panic(err)
	}

	if upgradeInfo.Name == UpgradeName && !app.UpgradeKeeper.IsSkipHeight(upgradeInfo.Height) {
		storeUpgrades := storetypes.StoreUpgrades{
			Added: []string{},
		}
		// configure store loader that checks if version == upgradeHeight and applies store upgrades
		app.SetStoreLoader(upgradetypes.UpgradeStoreLoader(upgradeInfo.Height, &storeUpgrades))
	}
}

// migrateLegacyAddressPrefixes rewrites the bech32-encoded address strings
// stored in state from the legacy upstream "cosmos" prefixes to the prefixes
// of this chain.
//
// Accounts and staking objects persist their addresses as bech32 strings.
// Chains bootstrapped with the upstream cosmos/evm binaries (v0.5.x) stored
// every address with the "cosmos" prefixes; since this chain uses its own
// prefixes, those legacy strings no longer decode and e.g.
// BaseAccount.GetAddress() silently resolves to the empty address, breaking
// module account operations such as minting. The underlying address bytes are
// unchanged: only the embedded string representation is re-encoded, so every
// store key derived from these addresses keeps its original value.
func (app EVMD) migrateLegacyAddressPrefixes(ctx context.Context) error {
	cfg := sdk.GetConfig()
	legacyPrefixes := map[string]string{
		"cosmos":        cfg.GetBech32AccountAddrPrefix(),
		"cosmosvaloper": cfg.GetBech32ValidatorAddrPrefix(),
	}

	// convert re-encodes a bech32 address that uses a legacy prefix with the
	// prefix this chain expects. Values that are not legacy-prefixed bech32
	// addresses are returned unchanged.
	convert := func(addr string) (string, bool, error) {
		hrp, bz, err := bech32.DecodeAndConvert(addr)
		if err != nil {
			return addr, false, nil
		}
		newPrefix, ok := legacyPrefixes[hrp]
		if !ok {
			return addr, false, nil
		}
		out, err := bech32.ConvertAndEncode(newPrefix, bz)
		if err != nil {
			return addr, false, err
		}
		return out, true, nil
	}

	var migrateErr error
	fail := func(err error) bool {
		if migrateErr == nil {
			migrateErr = err
		}
		return true // stop iteration
	}

	// Auth accounts: the address string is embedded in every account proto.
	// SetAccount derives the store key from the account address, so the field
	// is updated first to keep the original key bytes.
	type accountUpdate struct {
		account sdk.AccountI
		newAddr string
	}
	var accountUpdates []accountUpdate
	app.AccountKeeper.IterateAccounts(ctx, func(acc sdk.AccountI) (stop bool) {
		ba := embeddedBaseAccount(acc)
		if ba == nil {
			return false
		}
		newAddr, changed, err := convert(ba.Address)
		if err != nil {
			return fail(err)
		}
		if !changed {
			return false
		}
		accountUpdates = append(accountUpdates, accountUpdate{account: acc, newAddr: newAddr})
		return false
	})
	if migrateErr != nil {
		return migrateErr
	}
	for _, u := range accountUpdates {
		embeddedBaseAccount(u.account).Address = u.newAddr
		app.AccountKeeper.SetAccount(ctx, u.account)
	}

	// Staking objects: validators, delegations, unbonding delegations and
	// redelegations persist delegator and operator addresses as bech32
	// strings, and their keepers derive the store keys from those strings.
	var validatorUpdates []stakingtypes.Validator
	err := app.StakingKeeper.IterateValidators(ctx, func(_ int64, valI stakingtypes.ValidatorI) (stop bool) {
		val, ok := valI.(stakingtypes.Validator)
		if !ok {
			return false
		}
		newAddr, changed, err := convert(val.OperatorAddress)
		if err != nil {
			return fail(err)
		}
		if !changed {
			return false
		}
		val.OperatorAddress = newAddr
		validatorUpdates = append(validatorUpdates, val)
		return false
	})
	if err != nil {
		return err
	}
	if migrateErr != nil {
		return migrateErr
	}
	for _, val := range validatorUpdates {
		if err := app.StakingKeeper.SetValidator(ctx, val); err != nil {
			return err
		}
	}

	var delegationUpdates []stakingtypes.Delegation
	err = app.StakingKeeper.IterateAllDelegations(ctx, func(d stakingtypes.Delegation) (stop bool) {
		newDelegator, changedDelegator, err := convert(d.DelegatorAddress)
		if err != nil {
			return fail(err)
		}
		newValidator, changedValidator, err := convert(d.ValidatorAddress)
		if err != nil {
			return fail(err)
		}
		if !changedDelegator && !changedValidator {
			return false
		}
		d.DelegatorAddress = newDelegator
		d.ValidatorAddress = newValidator
		delegationUpdates = append(delegationUpdates, d)
		return false
	})
	if err != nil {
		return err
	}
	if migrateErr != nil {
		return migrateErr
	}
	for _, d := range delegationUpdates {
		if err := app.StakingKeeper.SetDelegation(ctx, d); err != nil {
			return err
		}
	}

	var ubdUpdates []stakingtypes.UnbondingDelegation
	err = app.StakingKeeper.IterateUnbondingDelegations(ctx, func(_ int64, ubd stakingtypes.UnbondingDelegation) (stop bool) {
		newDelegator, changedDelegator, err := convert(ubd.DelegatorAddress)
		if err != nil {
			return fail(err)
		}
		newValidator, changedValidator, err := convert(ubd.ValidatorAddress)
		if err != nil {
			return fail(err)
		}
		if !changedDelegator && !changedValidator {
			return false
		}
		ubd.DelegatorAddress = newDelegator
		ubd.ValidatorAddress = newValidator
		ubdUpdates = append(ubdUpdates, ubd)
		return false
	})
	if err != nil {
		return err
	}
	if migrateErr != nil {
		return migrateErr
	}
	for _, ubd := range ubdUpdates {
		if err := app.StakingKeeper.SetUnbondingDelegation(ctx, ubd); err != nil {
			return err
		}
	}

	var redUpdates []stakingtypes.Redelegation
	err = app.StakingKeeper.IterateRedelegations(ctx, func(_ int64, red stakingtypes.Redelegation) (stop bool) {
		newDelegator, changedDelegator, err := convert(red.DelegatorAddress)
		if err != nil {
			return fail(err)
		}
		newSrc, changedSrc, err := convert(red.ValidatorSrcAddress)
		if err != nil {
			return fail(err)
		}
		newDst, changedDst, err := convert(red.ValidatorDstAddress)
		if err != nil {
			return fail(err)
		}
		if !changedDelegator && !changedSrc && !changedDst {
			return false
		}
		red.DelegatorAddress = newDelegator
		red.ValidatorSrcAddress = newSrc
		red.ValidatorDstAddress = newDst
		redUpdates = append(redUpdates, red)
		return false
	})
	if err != nil {
		return err
	}
	if migrateErr != nil {
		return migrateErr
	}
	for _, red := range redUpdates {
		if err := app.StakingKeeper.SetRedelegation(ctx, red); err != nil {
			return err
		}
	}

	return nil
}

// embeddedBaseAccount returns the BaseAccount embedded in the given account,
// or nil for account types that do not expose one.
func embeddedBaseAccount(acc sdk.AccountI) *authtypes.BaseAccount {
	switch a := acc.(type) {
	case *authtypes.BaseAccount:
		return a
	case *authtypes.ModuleAccount:
		return a.BaseAccount
	case *vestingtypes.BaseVestingAccount:
		return a.BaseAccount
	case *vestingtypes.ContinuousVestingAccount:
		return a.BaseVestingAccount.BaseAccount
	case *vestingtypes.DelayedVestingAccount:
		return a.BaseVestingAccount.BaseAccount
	case *vestingtypes.PeriodicVestingAccount:
		return a.BaseVestingAccount.BaseAccount
	case *vestingtypes.PermanentLockedAccount:
		return a.BaseVestingAccount.BaseAccount
	default:
		return nil
	}
}

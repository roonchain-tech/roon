package ibc

import (
	"errors"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"

	"github.com/cosmos/evm"
	"github.com/cosmos/evm/contracts"
	evmibctesting "github.com/cosmos/evm/testutil/ibc"
	testutiltypes "github.com/cosmos/evm/testutil/types"
	erc20types "github.com/cosmos/evm/x/erc20/types"
	ibctesting "github.com/cosmos/ibc-go/v10/testing"

	errorsmod "cosmossdk.io/errors"

	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
)

// The counterparty Cosmos chains in these IBC tests are built with the ibc-go
// simapp, whose keepers hardcode the default SDK bech32 prefixes ("cosmos").
// The chain test helpers set a custom prefix ("roon") globally, which makes
// module authority addresses fail the simapp codec validation and breaks tx
// signing on the counterparty chains. The whole test binary must therefore run
// with the default SDK prefixes, matching upstream where the chain prefix is
// "cosmos".
func init() {
	cfg := sdk.GetConfig()
	cfg.SetBech32PrefixForAccount(sdk.Bech32MainPrefix, sdk.Bech32MainPrefix+sdk.PrefixPublic)
	cfg.SetBech32PrefixForValidator(sdk.Bech32MainPrefix+sdk.PrefixValidator+sdk.PrefixOperator, sdk.Bech32MainPrefix+sdk.PrefixValidator+sdk.PrefixOperator+sdk.PrefixPublic)
	cfg.SetBech32PrefixForConsensusNode(sdk.Bech32MainPrefix+sdk.PrefixValidator+sdk.PrefixConsensus, sdk.Bech32MainPrefix+sdk.PrefixValidator+sdk.PrefixConsensus+sdk.PrefixPublic)
}

// NativeErc20Info holds details about a deployed ERC20 token.
type NativeErc20Info struct {
	Denom        string
	ContractAbi  abi.ABI
	ContractAddr common.Address
	Account      common.Address // The address of the minter on the EVM chain
	InitialBal   *big.Int
}

// SetupNativeErc20 deploys, registers, and mints a native ERC20 token on an EVM-based chain.
func SetupNativeErc20(t *testing.T, chain *evmibctesting.TestChain, senderAcc evmibctesting.SenderAccount) *NativeErc20Info {
	t.Helper()

	evmCtx := chain.GetContext()
	evmApp := chain.App.(evm.EvmApp)

	// Deploy new ERC20 contract with default metadata
	contractAddr, err := evmApp.GetErc20Keeper().DeployERC20Contract(evmCtx, banktypes.Metadata{
		DenomUnits: []*banktypes.DenomUnit{
			{Denom: "example", Exponent: 18},
		},
		Name:   "Example",
		Symbol: "Ex",
	})
	if err != nil {
		t.Fatalf("ERC20 deployment failed: %v", err)
	}
	chain.NextBlock()

	// Register the contract
	_, err = evmApp.GetErc20Keeper().RegisterERC20(evmCtx, &erc20types.MsgRegisterERC20{
		Signer:         authtypes.NewModuleAddress(govtypes.ModuleName).String(), // does not have to be gov
		Erc20Addresses: []string{contractAddr.Hex()},
	})
	if err != nil {
		t.Fatalf("RegisterERC20 failed: %v", err)
	}

	// Mint tokens to default sender
	contractAbi := contracts.ERC20MinterBurnerDecimalsContract.ABI
	nativeDenom := erc20types.CreateDenom(contractAddr.String())
	sendAmt := ibctesting.DefaultCoinAmount
	senderAddr := senderAcc.SenderAccount.GetAddress()

	_, err = evmApp.GetEVMKeeper().CallEVM(
		evmCtx,
		contractAbi,
		erc20types.ModuleAddress,
		contractAddr,
		true,
		nil,
		"mint",
		common.BytesToAddress(senderAddr),
		big.NewInt(sendAmt.Int64()),
	)
	if err != nil {
		t.Fatalf("mint call failed: %v", err)
	}

	// Verify minted balance
	ethRes, err := evmApp.GetEVMKeeper().CallEVM(
		evmCtx,
		contractAbi,
		common.BytesToAddress(senderAddr),
		contractAddr,
		false,
		nil,
		"balanceOf",
		common.BytesToAddress(senderAddr),
	)
	if err != nil {
		t.Fatalf("balanceOf call failed: %v", err)
	}
	var bal *big.Int
	err = contractAbi.UnpackIntoInterface(&bal, "balanceOf", ethRes.Ret)
	if err != nil {
		t.Fatalf("balanceOf call failed: %v", err)
	}
	if bal.Cmp(big.NewInt(sendAmt.Int64())) != 0 {
		t.Fatalf("unexpected ERC20 balance; got %s, want %s", bal.String(), sendAmt.String())
	}

	return &NativeErc20Info{
		Denom:        nativeDenom,
		ContractAbi:  contractAbi,
		ContractAddr: contractAddr,
		Account:      common.BytesToAddress(senderAddr),
		InitialBal:   big.NewInt(sendAmt.Int64()),
	}
}

// SetupNativeErc20 deploys, registers, and mints a native ERC20 token on an EVM-based chain.
func DeployContract(t *testing.T, chain *evmibctesting.TestChain, deploymentData testutiltypes.ContractDeploymentData) (common.Address, error) {
	t.Helper()

	// Get account's nonce to create contract hash
	from := common.BytesToAddress(chain.SenderPrivKey.PubKey().Address().Bytes())
	account := chain.App.(evm.EvmApp).GetEVMKeeper().GetAccount(chain.GetContext(), from)
	if account == nil {
		return common.Address{}, errors.New("account not found")
	}

	ctorArgs, err := deploymentData.Contract.ABI.Pack("", deploymentData.ConstructorArgs...)
	if err != nil {
		return common.Address{}, errorsmod.Wrap(err, "failed to pack constructor arguments")
	}

	data := deploymentData.Contract.Bin
	data = append(data, ctorArgs...)

	_, err = chain.App.(evm.EvmApp).GetEVMKeeper().CallEVMWithData(chain.GetContext(), from, nil, data, true, nil)
	if err != nil {
		return common.Address{}, errorsmod.Wrapf(err, "failed to deploy contract")
	}

	return crypto.CreateAddress(from, account.Nonce), nil
}

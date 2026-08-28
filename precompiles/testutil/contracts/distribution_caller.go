package contracts

import (
	contractutils "github.com/cosmos/evm/contracts/utils"
	evmtypes "github.com/cosmos/evm/x/vm/types"
)

func LoadDistributionCallerContract() (evmtypes.CompiledContract, error) {
	return contractutils.LoadContractFromJSONFile("DistributionCaller.json")
}

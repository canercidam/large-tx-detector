package contracts

import (
	"strings"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

// BindIERC20 binds a generic wrapper to an already deployed contract.
func BindIERC20(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := abi.JSON(strings.NewReader(IERC20ABI))
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, parsed, caller, transactor, filterer), nil
}

// UnpackIERC20Transfer unpacks the event.
func UnpackIERC20Transfer(contract *bind.BoundContract, log *types.Log) (*IERC20Transfer, error) {
	var transferEvent IERC20Transfer
	return &transferEvent, contract.UnpackLog(&transferEvent, "Transfer", *log)
}

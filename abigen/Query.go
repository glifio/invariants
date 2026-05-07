// Code generated - DO NOT EDIT.
// This file is a generated binding and any manual changes will be lost.

package abigen

import (
	"errors"
	"math/big"
	"strings"

	ethereum "github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/event"
)

// Reference imports to suppress errors if they are not otherwise used.
var (
	_ = errors.New
	_ = big.NewInt
	_ = strings.NewReader
	_ = ethereum.NotFound
	_ = bind.Bind
	_ = common.Big1
	_ = types.BloomLookup
	_ = event.NewSubscription
	_ = abi.ConvertType
)

// QueryMetaData contains all meta data concerning the Query contract.
var QueryMetaData = &bind.MetaData{
	ABI: "[{\"inputs\":[{\"internalType\":\"address\",\"name\":\"router_\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"ifil_\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"rm_\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"pool_\",\"type\":\"address\"}],\"stateMutability\":\"nonpayable\",\"type\":\"constructor\"},{\"inputs\":[{\"internalType\":\"uint32[]\",\"name\":\"agentIDs\",\"type\":\"uint32[]\"}],\"name\":\"getAgentInterestOwed\",\"outputs\":[{\"internalType\":\"uint256[]\",\"name\":\"interestOwed\",\"type\":\"uint256[]\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address[]\",\"name\":\"agentAddrs\",\"type\":\"address[]\"}],\"name\":\"getAgentOwners\",\"outputs\":[{\"internalType\":\"address[]\",\"name\":\"owners\",\"type\":\"address[]\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address[]\",\"name\":\"agentAddrs\",\"type\":\"address[]\"}],\"name\":\"getAgentsEpochsPaid\",\"outputs\":[{\"internalType\":\"uint256[]\",\"name\":\"epochsPaid\",\"type\":\"uint256[]\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint32[]\",\"name\":\"agentIDs\",\"type\":\"uint32[]\"}],\"name\":\"getAgentsLevels\",\"outputs\":[{\"internalType\":\"uint256[]\",\"name\":\"levels\",\"type\":\"uint256[]\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address[]\",\"name\":\"agentAddrs\",\"type\":\"address[]\"}],\"name\":\"getAgentsLiquidAssets\",\"outputs\":[{\"internalType\":\"uint256[]\",\"name\":\"liquidAssets\",\"type\":\"uint256[]\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address[]\",\"name\":\"depositorAddrs\",\"type\":\"address[]\"}],\"name\":\"getDepositorsIFILBalances\",\"outputs\":[{\"internalType\":\"uint256[]\",\"name\":\"ifilBalances\",\"type\":\"uint256[]\"}],\"stateMutability\":\"view\",\"type\":\"function\"}]",
	Bin: "0x61010060405273c29aa37c00b01ceaf3202bff47295fc579e920fe60a05273690908f7fa93afc040cfbd9fe1ddd2c2668aa0e060c05273e764acf02d8b7c21d2b6a8f0a96c78541e0dc3fd60e052348015610058575f80fd5b50604051610db4380380610db4833981016040819052610077916100f8565b6001600160a01b0380851660805283161561009a576001600160a01b03831660c0525b6001600160a01b038216156100b7576001600160a01b03821660a0525b6001600160a01b038116156100d4576001600160a01b03811660e0525b50505050610149565b80516001600160a01b03811681146100f3575f80fd5b919050565b5f805f806080858703121561010b575f80fd5b610114856100dd565b9350610122602086016100dd565b9250610130604086016100dd565b915061013e606086016100dd565b905092959194509250565b60805160a05160c05160e051610c3861017c5f395f6102a601525f6106ad01525f61053c01525f6108650152610c385ff3fe608060405234801561000f575f80fd5b506004361061006e575f3560e01c8063a9acbb4c1161004d578063a9acbb4c146100ce578063bd699198146100e1578063ea4d65df146100f4575f80fd5b80620bf1d814610072578063064fcfd81461009b578063a35c3810146100bb575b5f80fd5b610085610080366004610964565b610107565b60405161009291906109a3565b60405180910390f35b6100ae6100a9366004610964565b61024d565b60405161009291906109fc565b6100ae6100c9366004610964565b6103be565b6100ae6100dc366004610964565b6104e3565b6100ae6100ef366004610964565b610654565b6100ae610102366004610964565b6107d4565b60608167ffffffffffffffff81111561012257610122610a33565b60405190808252806020026020018201604052801561014b578160200160208202803683370190505b5090505f5b63ffffffff81168311156102465783838263ffffffff1681811061017657610176610a60565b905060200201602081019061018b9190610ab1565b73ffffffffffffffffffffffffffffffffffffffff16638da5cb5b6040518163ffffffff1660e01b8152600401602060405180830381865afa1580156101d3573d5f803e3d5ffd5b505050506040513d601f19601f820116820180604052508101906101f79190610ad3565b828263ffffffff168151811061020f5761020f610a60565b73ffffffffffffffffffffffffffffffffffffffff909216602092830291909101909101528061023e81610aee565b915050610150565b5092915050565b60608167ffffffffffffffff81111561026857610268610a33565b604051908082528060200260200182016040528015610291578160200160208202803683370190505b5090505f5b63ffffffff8116831115610246577f000000000000000000000000000000000000000000000000000000000000000073ffffffffffffffffffffffffffffffffffffffff1663f086ce6085858463ffffffff168181106102f8576102f8610a60565b905060200201602081019061030d9190610b35565b6040517fffffffff0000000000000000000000000000000000000000000000000000000060e084901b16815263ffffffff919091166004820152602401602060405180830381865afa158015610365573d5f803e3d5ffd5b505050506040513d601f19601f820116820180604052508101906103899190610b58565b828263ffffffff16815181106103a1576103a1610a60565b6020908102919091010152806103b681610aee565b915050610296565b60608167ffffffffffffffff8111156103d9576103d9610a33565b604051908082528060200260200182016040528015610402578160200160208202803683370190505b5090505f5b63ffffffff81168311156102465783838263ffffffff1681811061042d5761042d610a60565b90506020020160208101906104429190610ab1565b73ffffffffffffffffffffffffffffffffffffffff1663e492cdce6040518163ffffffff1660e01b8152600401602060405180830381865afa15801561048a573d5f803e3d5ffd5b505050506040513d601f19601f820116820180604052508101906104ae9190610b58565b828263ffffffff16815181106104c6576104c6610a60565b6020908102919091010152806104db81610aee565b915050610407565b60608167ffffffffffffffff8111156104fe576104fe610a33565b604051908082528060200260200182016040528015610527578160200160208202803683370190505b5090505f5b63ffffffff8116831115610246577f000000000000000000000000000000000000000000000000000000000000000073ffffffffffffffffffffffffffffffffffffffff16639c18625f85858463ffffffff1681811061058e5761058e610a60565b90506020020160208101906105a39190610b35565b6040517fffffffff0000000000000000000000000000000000000000000000000000000060e084901b16815263ffffffff919091166004820152602401602060405180830381865afa1580156105fb573d5f803e3d5ffd5b505050506040513d601f19601f8201168201806040525081019061061f9190610b58565b828263ffffffff168151811061063757610637610a60565b60209081029190910101528061064c81610aee565b91505061052c565b60608167ffffffffffffffff81111561066f5761066f610a33565b604051908082528060200260200182016040528015610698578160200160208202803683370190505b5090505f5b63ffffffff8116831115610246577f000000000000000000000000000000000000000000000000000000000000000073ffffffffffffffffffffffffffffffffffffffff166370a0823185858463ffffffff168181106106ff576106ff610a60565b90506020020160208101906107149190610ab1565b6040517fffffffff0000000000000000000000000000000000000000000000000000000060e084901b16815273ffffffffffffffffffffffffffffffffffffffff9091166004820152602401602060405180830381865afa15801561077b573d5f803e3d5ffd5b505050506040513d601f19601f8201168201806040525081019061079f9190610b58565b828263ffffffff16815181106107b7576107b7610a60565b6020908102919091010152806107cc81610aee565b91505061069d565b60608167ffffffffffffffff8111156107ef576107ef610a33565b604051908082528060200260200182016040528015610818578160200160208202803683370190505b5090505f5b63ffffffff8116831115610246576040517f6361f6de00000000000000000000000000000000000000000000000000000000815263ffffffff821660048201525f60248201527f000000000000000000000000000000000000000000000000000000000000000073ffffffffffffffffffffffffffffffffffffffff1690636361f6de90604401608060405180830381865afa1580156108bf573d5f803e3d5ffd5b505050506040513d601f19601f820116820180604052508101906108e39190610b6f565b60400151828263ffffffff16815181106108ff576108ff610a60565b60209081029190910101528061091481610aee565b91505061081d565b5f8083601f84011261092c575f80fd5b50813567ffffffffffffffff811115610943575f80fd5b6020830191508360208260051b850101111561095d575f80fd5b9250929050565b5f8060208385031215610975575f80fd5b823567ffffffffffffffff81111561098b575f80fd5b6109978582860161091c565b90969095509350505050565b602080825282518282018190525f9190848201906040850190845b818110156109f057835173ffffffffffffffffffffffffffffffffffffffff16835292840192918401916001016109be565b50909695505050505050565b602080825282518282018190525f9190848201906040850190845b818110156109f057835183529284019291840191600101610a17565b7f4e487b71000000000000000000000000000000000000000000000000000000005f52604160045260245ffd5b7f4e487b71000000000000000000000000000000000000000000000000000000005f52603260045260245ffd5b73ffffffffffffffffffffffffffffffffffffffff81168114610aae575f80fd5b50565b5f60208284031215610ac1575f80fd5b8135610acc81610a8d565b9392505050565b5f60208284031215610ae3575f80fd5b8151610acc81610a8d565b5f63ffffffff808316818103610b2b577f4e487b71000000000000000000000000000000000000000000000000000000005f52601160045260245ffd5b6001019392505050565b5f60208284031215610b45575f80fd5b813563ffffffff81168114610acc575f80fd5b5f60208284031215610b68575f80fd5b5051919050565b5f60808284031215610b7f575f80fd5b6040516080810181811067ffffffffffffffff82111715610bc7577f4e487b71000000000000000000000000000000000000000000000000000000005f52604160045260245ffd5b806040525082518152602083015160208201526040830151604082015260608301518015158114610bf6575f80fd5b6060820152939250505056fea264697066735822122054a7272d1fbc66d1a8aad145d0975ff5bd172ca3132f066ab38437145cc41f8364736f6c63430008150033",
}

// QueryABI is the input ABI used to generate the binding from.
// Deprecated: Use QueryMetaData.ABI instead.
var QueryABI = QueryMetaData.ABI

// QueryBin is the compiled bytecode used for deploying new contracts.
// Deprecated: Use QueryMetaData.Bin instead.
var QueryBin = QueryMetaData.Bin

// DeployQuery deploys a new Ethereum contract, binding an instance of Query to it.
func DeployQuery(auth *bind.TransactOpts, backend bind.ContractBackend, router_ common.Address, ifil_ common.Address, rm_ common.Address, pool_ common.Address) (common.Address, *types.Transaction, *Query, error) {
	parsed, err := QueryMetaData.GetAbi()
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	if parsed == nil {
		return common.Address{}, nil, nil, errors.New("GetABI returned nil")
	}

	address, tx, contract, err := bind.DeployContract(auth, *parsed, common.FromHex(QueryBin), backend, router_, ifil_, rm_, pool_)
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	return address, tx, &Query{QueryCaller: QueryCaller{contract: contract}, QueryTransactor: QueryTransactor{contract: contract}, QueryFilterer: QueryFilterer{contract: contract}}, nil
}

// Query is an auto generated Go binding around an Ethereum contract.
type Query struct {
	QueryCaller     // Read-only binding to the contract
	QueryTransactor // Write-only binding to the contract
	QueryFilterer   // Log filterer for contract events
}

// QueryCaller is an auto generated read-only Go binding around an Ethereum contract.
type QueryCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// QueryTransactor is an auto generated write-only Go binding around an Ethereum contract.
type QueryTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// QueryFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type QueryFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// QuerySession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type QuerySession struct {
	Contract     *Query            // Generic contract binding to set the session for
	CallOpts     bind.CallOpts     // Call options to use throughout this session
	TransactOpts bind.TransactOpts // Transaction auth options to use throughout this session
}

// QueryCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type QueryCallerSession struct {
	Contract *QueryCaller  // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts // Call options to use throughout this session
}

// QueryTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type QueryTransactorSession struct {
	Contract     *QueryTransactor  // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts // Transaction auth options to use throughout this session
}

// QueryRaw is an auto generated low-level Go binding around an Ethereum contract.
type QueryRaw struct {
	Contract *Query // Generic contract binding to access the raw methods on
}

// QueryCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type QueryCallerRaw struct {
	Contract *QueryCaller // Generic read-only contract binding to access the raw methods on
}

// QueryTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type QueryTransactorRaw struct {
	Contract *QueryTransactor // Generic write-only contract binding to access the raw methods on
}

// NewQuery creates a new instance of Query, bound to a specific deployed contract.
func NewQuery(address common.Address, backend bind.ContractBackend) (*Query, error) {
	contract, err := bindQuery(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &Query{QueryCaller: QueryCaller{contract: contract}, QueryTransactor: QueryTransactor{contract: contract}, QueryFilterer: QueryFilterer{contract: contract}}, nil
}

// NewQueryCaller creates a new read-only instance of Query, bound to a specific deployed contract.
func NewQueryCaller(address common.Address, caller bind.ContractCaller) (*QueryCaller, error) {
	contract, err := bindQuery(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &QueryCaller{contract: contract}, nil
}

// NewQueryTransactor creates a new write-only instance of Query, bound to a specific deployed contract.
func NewQueryTransactor(address common.Address, transactor bind.ContractTransactor) (*QueryTransactor, error) {
	contract, err := bindQuery(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &QueryTransactor{contract: contract}, nil
}

// NewQueryFilterer creates a new log filterer instance of Query, bound to a specific deployed contract.
func NewQueryFilterer(address common.Address, filterer bind.ContractFilterer) (*QueryFilterer, error) {
	contract, err := bindQuery(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &QueryFilterer{contract: contract}, nil
}

// bindQuery binds a generic wrapper to an already deployed contract.
func bindQuery(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := QueryMetaData.GetAbi()
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, *parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_Query *QueryRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _Query.Contract.QueryCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_Query *QueryRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _Query.Contract.QueryTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_Query *QueryRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _Query.Contract.QueryTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_Query *QueryCallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _Query.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_Query *QueryTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _Query.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_Query *QueryTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _Query.Contract.contract.Transact(opts, method, params...)
}

// GetAgentInterestOwed is a free data retrieval call binding the contract method 0x064fcfd8.
//
// Solidity: function getAgentInterestOwed(uint32[] agentIDs) view returns(uint256[] interestOwed)
func (_Query *QueryCaller) GetAgentInterestOwed(opts *bind.CallOpts, agentIDs []uint32) ([]*big.Int, error) {
	var out []interface{}
	err := _Query.contract.Call(opts, &out, "getAgentInterestOwed", agentIDs)

	if err != nil {
		return *new([]*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new([]*big.Int)).(*[]*big.Int)

	return out0, err

}

// GetAgentInterestOwed is a free data retrieval call binding the contract method 0x064fcfd8.
//
// Solidity: function getAgentInterestOwed(uint32[] agentIDs) view returns(uint256[] interestOwed)
func (_Query *QuerySession) GetAgentInterestOwed(agentIDs []uint32) ([]*big.Int, error) {
	return _Query.Contract.GetAgentInterestOwed(&_Query.CallOpts, agentIDs)
}

// GetAgentInterestOwed is a free data retrieval call binding the contract method 0x064fcfd8.
//
// Solidity: function getAgentInterestOwed(uint32[] agentIDs) view returns(uint256[] interestOwed)
func (_Query *QueryCallerSession) GetAgentInterestOwed(agentIDs []uint32) ([]*big.Int, error) {
	return _Query.Contract.GetAgentInterestOwed(&_Query.CallOpts, agentIDs)
}

// GetAgentOwners is a free data retrieval call binding the contract method 0x000bf1d8.
//
// Solidity: function getAgentOwners(address[] agentAddrs) view returns(address[] owners)
func (_Query *QueryCaller) GetAgentOwners(opts *bind.CallOpts, agentAddrs []common.Address) ([]common.Address, error) {
	var out []interface{}
	err := _Query.contract.Call(opts, &out, "getAgentOwners", agentAddrs)

	if err != nil {
		return *new([]common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new([]common.Address)).(*[]common.Address)

	return out0, err

}

// GetAgentOwners is a free data retrieval call binding the contract method 0x000bf1d8.
//
// Solidity: function getAgentOwners(address[] agentAddrs) view returns(address[] owners)
func (_Query *QuerySession) GetAgentOwners(agentAddrs []common.Address) ([]common.Address, error) {
	return _Query.Contract.GetAgentOwners(&_Query.CallOpts, agentAddrs)
}

// GetAgentOwners is a free data retrieval call binding the contract method 0x000bf1d8.
//
// Solidity: function getAgentOwners(address[] agentAddrs) view returns(address[] owners)
func (_Query *QueryCallerSession) GetAgentOwners(agentAddrs []common.Address) ([]common.Address, error) {
	return _Query.Contract.GetAgentOwners(&_Query.CallOpts, agentAddrs)
}

// GetAgentsEpochsPaid is a free data retrieval call binding the contract method 0xea4d65df.
//
// Solidity: function getAgentsEpochsPaid(address[] agentAddrs) view returns(uint256[] epochsPaid)
func (_Query *QueryCaller) GetAgentsEpochsPaid(opts *bind.CallOpts, agentAddrs []common.Address) ([]*big.Int, error) {
	var out []interface{}
	err := _Query.contract.Call(opts, &out, "getAgentsEpochsPaid", agentAddrs)

	if err != nil {
		return *new([]*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new([]*big.Int)).(*[]*big.Int)

	return out0, err

}

// GetAgentsEpochsPaid is a free data retrieval call binding the contract method 0xea4d65df.
//
// Solidity: function getAgentsEpochsPaid(address[] agentAddrs) view returns(uint256[] epochsPaid)
func (_Query *QuerySession) GetAgentsEpochsPaid(agentAddrs []common.Address) ([]*big.Int, error) {
	return _Query.Contract.GetAgentsEpochsPaid(&_Query.CallOpts, agentAddrs)
}

// GetAgentsEpochsPaid is a free data retrieval call binding the contract method 0xea4d65df.
//
// Solidity: function getAgentsEpochsPaid(address[] agentAddrs) view returns(uint256[] epochsPaid)
func (_Query *QueryCallerSession) GetAgentsEpochsPaid(agentAddrs []common.Address) ([]*big.Int, error) {
	return _Query.Contract.GetAgentsEpochsPaid(&_Query.CallOpts, agentAddrs)
}

// GetAgentsLevels is a free data retrieval call binding the contract method 0xa9acbb4c.
//
// Solidity: function getAgentsLevels(uint32[] agentIDs) view returns(uint256[] levels)
func (_Query *QueryCaller) GetAgentsLevels(opts *bind.CallOpts, agentIDs []uint32) ([]*big.Int, error) {
	var out []interface{}
	err := _Query.contract.Call(opts, &out, "getAgentsLevels", agentIDs)

	if err != nil {
		return *new([]*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new([]*big.Int)).(*[]*big.Int)

	return out0, err

}

// GetAgentsLevels is a free data retrieval call binding the contract method 0xa9acbb4c.
//
// Solidity: function getAgentsLevels(uint32[] agentIDs) view returns(uint256[] levels)
func (_Query *QuerySession) GetAgentsLevels(agentIDs []uint32) ([]*big.Int, error) {
	return _Query.Contract.GetAgentsLevels(&_Query.CallOpts, agentIDs)
}

// GetAgentsLevels is a free data retrieval call binding the contract method 0xa9acbb4c.
//
// Solidity: function getAgentsLevels(uint32[] agentIDs) view returns(uint256[] levels)
func (_Query *QueryCallerSession) GetAgentsLevels(agentIDs []uint32) ([]*big.Int, error) {
	return _Query.Contract.GetAgentsLevels(&_Query.CallOpts, agentIDs)
}

// GetAgentsLiquidAssets is a free data retrieval call binding the contract method 0xa35c3810.
//
// Solidity: function getAgentsLiquidAssets(address[] agentAddrs) view returns(uint256[] liquidAssets)
func (_Query *QueryCaller) GetAgentsLiquidAssets(opts *bind.CallOpts, agentAddrs []common.Address) ([]*big.Int, error) {
	var out []interface{}
	err := _Query.contract.Call(opts, &out, "getAgentsLiquidAssets", agentAddrs)

	if err != nil {
		return *new([]*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new([]*big.Int)).(*[]*big.Int)

	return out0, err

}

// GetAgentsLiquidAssets is a free data retrieval call binding the contract method 0xa35c3810.
//
// Solidity: function getAgentsLiquidAssets(address[] agentAddrs) view returns(uint256[] liquidAssets)
func (_Query *QuerySession) GetAgentsLiquidAssets(agentAddrs []common.Address) ([]*big.Int, error) {
	return _Query.Contract.GetAgentsLiquidAssets(&_Query.CallOpts, agentAddrs)
}

// GetAgentsLiquidAssets is a free data retrieval call binding the contract method 0xa35c3810.
//
// Solidity: function getAgentsLiquidAssets(address[] agentAddrs) view returns(uint256[] liquidAssets)
func (_Query *QueryCallerSession) GetAgentsLiquidAssets(agentAddrs []common.Address) ([]*big.Int, error) {
	return _Query.Contract.GetAgentsLiquidAssets(&_Query.CallOpts, agentAddrs)
}

// GetDepositorsIFILBalances is a free data retrieval call binding the contract method 0xbd699198.
//
// Solidity: function getDepositorsIFILBalances(address[] depositorAddrs) view returns(uint256[] ifilBalances)
func (_Query *QueryCaller) GetDepositorsIFILBalances(opts *bind.CallOpts, depositorAddrs []common.Address) ([]*big.Int, error) {
	var out []interface{}
	err := _Query.contract.Call(opts, &out, "getDepositorsIFILBalances", depositorAddrs)

	if err != nil {
		return *new([]*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new([]*big.Int)).(*[]*big.Int)

	return out0, err

}

// GetDepositorsIFILBalances is a free data retrieval call binding the contract method 0xbd699198.
//
// Solidity: function getDepositorsIFILBalances(address[] depositorAddrs) view returns(uint256[] ifilBalances)
func (_Query *QuerySession) GetDepositorsIFILBalances(depositorAddrs []common.Address) ([]*big.Int, error) {
	return _Query.Contract.GetDepositorsIFILBalances(&_Query.CallOpts, depositorAddrs)
}

// GetDepositorsIFILBalances is a free data retrieval call binding the contract method 0xbd699198.
//
// Solidity: function getDepositorsIFILBalances(address[] depositorAddrs) view returns(uint256[] ifilBalances)
func (_Query *QueryCallerSession) GetDepositorsIFILBalances(depositorAddrs []common.Address) ([]*big.Int, error) {
	return _Query.Contract.GetDepositorsIFILBalances(&_Query.CallOpts, depositorAddrs)
}

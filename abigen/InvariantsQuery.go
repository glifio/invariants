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

// InvariantsQueryAgentDTLInputs is an auto generated low-level Go binding around an user-defined struct.
type InvariantsQueryAgentDTLInputs struct {
	LiquidAssets *big.Int
	MinerIDs     []uint64
	Principal    *big.Int
	Interest     *big.Int
}

// InvariantsQueryAgentState is an auto generated low-level Go binding around an user-defined struct.
type InvariantsQueryAgentState struct {
	StartEpoch   *big.Int
	Principal    *big.Int
	EpochsPaid   *big.Int
	InterestOwed *big.Int
	Defaulted    bool
}

// InvariantsQueryLPPlusTokenState is an auto generated low-level Go binding around an user-defined struct.
type InvariantsQueryLPPlusTokenState struct {
	Owner      common.Address
	RwtBalance *big.Int
	YbtBalance *big.Int
	IsActive   bool
}

// InvariantsQueryPoolMetrics is an auto generated low-level Go binding around an user-defined struct.
type InvariantsQueryPoolMetrics struct {
	TotalAssets      *big.Int
	TotalBorrowed    *big.Int
	TreasuryFeesOwed *big.Int
	IfilSupply       *big.Int
}

// InvariantsQuerySPPlusTokenState is an auto generated low-level Go binding around an user-defined struct.
type InvariantsQuerySPPlusTokenState struct {
	Owner   common.Address
	AgentId *big.Int
}

// InvariantsQueryMetaData contains all meta data concerning the InvariantsQuery contract.
var InvariantsQueryMetaData = &bind.MetaData{
	ABI: "[{\"type\":\"constructor\",\"inputs\":[{\"name\":\"_router\",\"type\":\"address\",\"internalType\":\"address\"},{\"name\":\"_pool\",\"type\":\"address\",\"internalType\":\"address\"},{\"name\":\"_ifil\",\"type\":\"address\",\"internalType\":\"address\"},{\"name\":\"_lpPlus\",\"type\":\"address\",\"internalType\":\"address\"},{\"name\":\"_spPlus\",\"type\":\"address\",\"internalType\":\"address\"},{\"name\":\"_minerRegistry\",\"type\":\"address\",\"internalType\":\"address\"}],\"stateMutability\":\"nonpayable\"},{\"type\":\"function\",\"name\":\"POOL_ID\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"uint256\",\"internalType\":\"uint256\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"getAgentsDTLInputs\",\"inputs\":[{\"name\":\"agentIDs\",\"type\":\"uint256[]\",\"internalType\":\"uint256[]\"},{\"name\":\"agentAddrs\",\"type\":\"address[]\",\"internalType\":\"address[]\"}],\"outputs\":[{\"name\":\"out\",\"type\":\"tuple[]\",\"internalType\":\"structInvariantsQuery.AgentDTLInputs[]\",\"components\":[{\"name\":\"liquidAssets\",\"type\":\"uint256\",\"internalType\":\"uint256\"},{\"name\":\"minerIDs\",\"type\":\"uint64[]\",\"internalType\":\"uint64[]\"},{\"name\":\"principal\",\"type\":\"uint256\",\"internalType\":\"uint256\"},{\"name\":\"interest\",\"type\":\"uint256\",\"internalType\":\"uint256\"}]}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"getAgentsState\",\"inputs\":[{\"name\":\"agentIDs\",\"type\":\"uint256[]\",\"internalType\":\"uint256[]\"}],\"outputs\":[{\"name\":\"out\",\"type\":\"tuple[]\",\"internalType\":\"structInvariantsQuery.AgentState[]\",\"components\":[{\"name\":\"startEpoch\",\"type\":\"uint256\",\"internalType\":\"uint256\"},{\"name\":\"principal\",\"type\":\"uint256\",\"internalType\":\"uint256\"},{\"name\":\"epochsPaid\",\"type\":\"uint256\",\"internalType\":\"uint256\"},{\"name\":\"interestOwed\",\"type\":\"uint256\",\"internalType\":\"uint256\"},{\"name\":\"defaulted\",\"type\":\"bool\",\"internalType\":\"bool\"}]}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"getLPPlusStates\",\"inputs\":[{\"name\":\"tokenIds\",\"type\":\"uint256[]\",\"internalType\":\"uint256[]\"}],\"outputs\":[{\"name\":\"out\",\"type\":\"tuple[]\",\"internalType\":\"structInvariantsQuery.LPPlusTokenState[]\",\"components\":[{\"name\":\"owner\",\"type\":\"address\",\"internalType\":\"address\"},{\"name\":\"rwtBalance\",\"type\":\"uint256\",\"internalType\":\"uint256\"},{\"name\":\"ybtBalance\",\"type\":\"uint256\",\"internalType\":\"uint256\"},{\"name\":\"isActive\",\"type\":\"bool\",\"internalType\":\"bool\"}]}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"getLPPlusTotalSupply\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"uint256\",\"internalType\":\"uint256\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"getPoolMetrics\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"tuple\",\"internalType\":\"structInvariantsQuery.PoolMetrics\",\"components\":[{\"name\":\"totalAssets\",\"type\":\"uint256\",\"internalType\":\"uint256\"},{\"name\":\"totalBorrowed\",\"type\":\"uint256\",\"internalType\":\"uint256\"},{\"name\":\"treasuryFeesOwed\",\"type\":\"uint256\",\"internalType\":\"uint256\"},{\"name\":\"ifilSupply\",\"type\":\"uint256\",\"internalType\":\"uint256\"}]}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"getSPPlusStates\",\"inputs\":[{\"name\":\"tokenIds\",\"type\":\"uint256[]\",\"internalType\":\"uint256[]\"}],\"outputs\":[{\"name\":\"out\",\"type\":\"tuple[]\",\"internalType\":\"structInvariantsQuery.SPPlusTokenState[]\",\"components\":[{\"name\":\"owner\",\"type\":\"address\",\"internalType\":\"address\"},{\"name\":\"agentId\",\"type\":\"uint256\",\"internalType\":\"uint256\"}]}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"getSPPlusTokenIdsForAgents\",\"inputs\":[{\"name\":\"agentIds\",\"type\":\"uint256[]\",\"internalType\":\"uint256[]\"}],\"outputs\":[{\"name\":\"out\",\"type\":\"uint256[]\",\"internalType\":\"uint256[]\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"ifil\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"address\",\"internalType\":\"contractIERC20\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"lpPlus\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"address\",\"internalType\":\"contractILPPlus\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"minerRegistry\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"address\",\"internalType\":\"contractIMinerRegistry\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"pool\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"address\",\"internalType\":\"contractIInfinityPoolV2\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"router\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"address\",\"internalType\":\"contractIRouter\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"spPlus\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"address\",\"internalType\":\"contractISPPlusV2\"}],\"stateMutability\":\"view\"}]",
}

// InvariantsQueryABI is the input ABI used to generate the binding from.
// Deprecated: Use InvariantsQueryMetaData.ABI instead.
var InvariantsQueryABI = InvariantsQueryMetaData.ABI

// InvariantsQuery is an auto generated Go binding around an Ethereum contract.
type InvariantsQuery struct {
	InvariantsQueryCaller     // Read-only binding to the contract
	InvariantsQueryTransactor // Write-only binding to the contract
	InvariantsQueryFilterer   // Log filterer for contract events
}

// InvariantsQueryCaller is an auto generated read-only Go binding around an Ethereum contract.
type InvariantsQueryCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// InvariantsQueryTransactor is an auto generated write-only Go binding around an Ethereum contract.
type InvariantsQueryTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// InvariantsQueryFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type InvariantsQueryFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// InvariantsQuerySession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type InvariantsQuerySession struct {
	Contract     *InvariantsQuery  // Generic contract binding to set the session for
	CallOpts     bind.CallOpts     // Call options to use throughout this session
	TransactOpts bind.TransactOpts // Transaction auth options to use throughout this session
}

// InvariantsQueryCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type InvariantsQueryCallerSession struct {
	Contract *InvariantsQueryCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts          // Call options to use throughout this session
}

// InvariantsQueryTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type InvariantsQueryTransactorSession struct {
	Contract     *InvariantsQueryTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts          // Transaction auth options to use throughout this session
}

// InvariantsQueryRaw is an auto generated low-level Go binding around an Ethereum contract.
type InvariantsQueryRaw struct {
	Contract *InvariantsQuery // Generic contract binding to access the raw methods on
}

// InvariantsQueryCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type InvariantsQueryCallerRaw struct {
	Contract *InvariantsQueryCaller // Generic read-only contract binding to access the raw methods on
}

// InvariantsQueryTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type InvariantsQueryTransactorRaw struct {
	Contract *InvariantsQueryTransactor // Generic write-only contract binding to access the raw methods on
}

// NewInvariantsQuery creates a new instance of InvariantsQuery, bound to a specific deployed contract.
func NewInvariantsQuery(address common.Address, backend bind.ContractBackend) (*InvariantsQuery, error) {
	contract, err := bindInvariantsQuery(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &InvariantsQuery{InvariantsQueryCaller: InvariantsQueryCaller{contract: contract}, InvariantsQueryTransactor: InvariantsQueryTransactor{contract: contract}, InvariantsQueryFilterer: InvariantsQueryFilterer{contract: contract}}, nil
}

// NewInvariantsQueryCaller creates a new read-only instance of InvariantsQuery, bound to a specific deployed contract.
func NewInvariantsQueryCaller(address common.Address, caller bind.ContractCaller) (*InvariantsQueryCaller, error) {
	contract, err := bindInvariantsQuery(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &InvariantsQueryCaller{contract: contract}, nil
}

// NewInvariantsQueryTransactor creates a new write-only instance of InvariantsQuery, bound to a specific deployed contract.
func NewInvariantsQueryTransactor(address common.Address, transactor bind.ContractTransactor) (*InvariantsQueryTransactor, error) {
	contract, err := bindInvariantsQuery(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &InvariantsQueryTransactor{contract: contract}, nil
}

// NewInvariantsQueryFilterer creates a new log filterer instance of InvariantsQuery, bound to a specific deployed contract.
func NewInvariantsQueryFilterer(address common.Address, filterer bind.ContractFilterer) (*InvariantsQueryFilterer, error) {
	contract, err := bindInvariantsQuery(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &InvariantsQueryFilterer{contract: contract}, nil
}

// bindInvariantsQuery binds a generic wrapper to an already deployed contract.
func bindInvariantsQuery(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := InvariantsQueryMetaData.GetAbi()
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, *parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_InvariantsQuery *InvariantsQueryRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _InvariantsQuery.Contract.InvariantsQueryCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_InvariantsQuery *InvariantsQueryRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _InvariantsQuery.Contract.InvariantsQueryTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_InvariantsQuery *InvariantsQueryRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _InvariantsQuery.Contract.InvariantsQueryTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_InvariantsQuery *InvariantsQueryCallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _InvariantsQuery.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_InvariantsQuery *InvariantsQueryTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _InvariantsQuery.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_InvariantsQuery *InvariantsQueryTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _InvariantsQuery.Contract.contract.Transact(opts, method, params...)
}

// POOLID is a free data retrieval call binding the contract method 0xe0d7d0e9.
//
// Solidity: function POOL_ID() view returns(uint256)
func (_InvariantsQuery *InvariantsQueryCaller) POOLID(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _InvariantsQuery.contract.Call(opts, &out, "POOL_ID")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// POOLID is a free data retrieval call binding the contract method 0xe0d7d0e9.
//
// Solidity: function POOL_ID() view returns(uint256)
func (_InvariantsQuery *InvariantsQuerySession) POOLID() (*big.Int, error) {
	return _InvariantsQuery.Contract.POOLID(&_InvariantsQuery.CallOpts)
}

// POOLID is a free data retrieval call binding the contract method 0xe0d7d0e9.
//
// Solidity: function POOL_ID() view returns(uint256)
func (_InvariantsQuery *InvariantsQueryCallerSession) POOLID() (*big.Int, error) {
	return _InvariantsQuery.Contract.POOLID(&_InvariantsQuery.CallOpts)
}

// GetAgentsDTLInputs is a free data retrieval call binding the contract method 0xc8a6f548.
//
// Solidity: function getAgentsDTLInputs(uint256[] agentIDs, address[] agentAddrs) view returns((uint256,uint64[],uint256,uint256)[] out)
func (_InvariantsQuery *InvariantsQueryCaller) GetAgentsDTLInputs(opts *bind.CallOpts, agentIDs []*big.Int, agentAddrs []common.Address) ([]InvariantsQueryAgentDTLInputs, error) {
	var out []interface{}
	err := _InvariantsQuery.contract.Call(opts, &out, "getAgentsDTLInputs", agentIDs, agentAddrs)

	if err != nil {
		return *new([]InvariantsQueryAgentDTLInputs), err
	}

	out0 := *abi.ConvertType(out[0], new([]InvariantsQueryAgentDTLInputs)).(*[]InvariantsQueryAgentDTLInputs)

	return out0, err

}

// GetAgentsDTLInputs is a free data retrieval call binding the contract method 0xc8a6f548.
//
// Solidity: function getAgentsDTLInputs(uint256[] agentIDs, address[] agentAddrs) view returns((uint256,uint64[],uint256,uint256)[] out)
func (_InvariantsQuery *InvariantsQuerySession) GetAgentsDTLInputs(agentIDs []*big.Int, agentAddrs []common.Address) ([]InvariantsQueryAgentDTLInputs, error) {
	return _InvariantsQuery.Contract.GetAgentsDTLInputs(&_InvariantsQuery.CallOpts, agentIDs, agentAddrs)
}

// GetAgentsDTLInputs is a free data retrieval call binding the contract method 0xc8a6f548.
//
// Solidity: function getAgentsDTLInputs(uint256[] agentIDs, address[] agentAddrs) view returns((uint256,uint64[],uint256,uint256)[] out)
func (_InvariantsQuery *InvariantsQueryCallerSession) GetAgentsDTLInputs(agentIDs []*big.Int, agentAddrs []common.Address) ([]InvariantsQueryAgentDTLInputs, error) {
	return _InvariantsQuery.Contract.GetAgentsDTLInputs(&_InvariantsQuery.CallOpts, agentIDs, agentAddrs)
}

// GetAgentsState is a free data retrieval call binding the contract method 0x94f7e000.
//
// Solidity: function getAgentsState(uint256[] agentIDs) view returns((uint256,uint256,uint256,uint256,bool)[] out)
func (_InvariantsQuery *InvariantsQueryCaller) GetAgentsState(opts *bind.CallOpts, agentIDs []*big.Int) ([]InvariantsQueryAgentState, error) {
	var out []interface{}
	err := _InvariantsQuery.contract.Call(opts, &out, "getAgentsState", agentIDs)

	if err != nil {
		return *new([]InvariantsQueryAgentState), err
	}

	out0 := *abi.ConvertType(out[0], new([]InvariantsQueryAgentState)).(*[]InvariantsQueryAgentState)

	return out0, err

}

// GetAgentsState is a free data retrieval call binding the contract method 0x94f7e000.
//
// Solidity: function getAgentsState(uint256[] agentIDs) view returns((uint256,uint256,uint256,uint256,bool)[] out)
func (_InvariantsQuery *InvariantsQuerySession) GetAgentsState(agentIDs []*big.Int) ([]InvariantsQueryAgentState, error) {
	return _InvariantsQuery.Contract.GetAgentsState(&_InvariantsQuery.CallOpts, agentIDs)
}

// GetAgentsState is a free data retrieval call binding the contract method 0x94f7e000.
//
// Solidity: function getAgentsState(uint256[] agentIDs) view returns((uint256,uint256,uint256,uint256,bool)[] out)
func (_InvariantsQuery *InvariantsQueryCallerSession) GetAgentsState(agentIDs []*big.Int) ([]InvariantsQueryAgentState, error) {
	return _InvariantsQuery.Contract.GetAgentsState(&_InvariantsQuery.CallOpts, agentIDs)
}

// GetLPPlusStates is a free data retrieval call binding the contract method 0xc878e522.
//
// Solidity: function getLPPlusStates(uint256[] tokenIds) view returns((address,uint256,uint256,bool)[] out)
func (_InvariantsQuery *InvariantsQueryCaller) GetLPPlusStates(opts *bind.CallOpts, tokenIds []*big.Int) ([]InvariantsQueryLPPlusTokenState, error) {
	var out []interface{}
	err := _InvariantsQuery.contract.Call(opts, &out, "getLPPlusStates", tokenIds)

	if err != nil {
		return *new([]InvariantsQueryLPPlusTokenState), err
	}

	out0 := *abi.ConvertType(out[0], new([]InvariantsQueryLPPlusTokenState)).(*[]InvariantsQueryLPPlusTokenState)

	return out0, err

}

// GetLPPlusStates is a free data retrieval call binding the contract method 0xc878e522.
//
// Solidity: function getLPPlusStates(uint256[] tokenIds) view returns((address,uint256,uint256,bool)[] out)
func (_InvariantsQuery *InvariantsQuerySession) GetLPPlusStates(tokenIds []*big.Int) ([]InvariantsQueryLPPlusTokenState, error) {
	return _InvariantsQuery.Contract.GetLPPlusStates(&_InvariantsQuery.CallOpts, tokenIds)
}

// GetLPPlusStates is a free data retrieval call binding the contract method 0xc878e522.
//
// Solidity: function getLPPlusStates(uint256[] tokenIds) view returns((address,uint256,uint256,bool)[] out)
func (_InvariantsQuery *InvariantsQueryCallerSession) GetLPPlusStates(tokenIds []*big.Int) ([]InvariantsQueryLPPlusTokenState, error) {
	return _InvariantsQuery.Contract.GetLPPlusStates(&_InvariantsQuery.CallOpts, tokenIds)
}

// GetLPPlusTotalSupply is a free data retrieval call binding the contract method 0x2a639cbe.
//
// Solidity: function getLPPlusTotalSupply() view returns(uint256)
func (_InvariantsQuery *InvariantsQueryCaller) GetLPPlusTotalSupply(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _InvariantsQuery.contract.Call(opts, &out, "getLPPlusTotalSupply")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// GetLPPlusTotalSupply is a free data retrieval call binding the contract method 0x2a639cbe.
//
// Solidity: function getLPPlusTotalSupply() view returns(uint256)
func (_InvariantsQuery *InvariantsQuerySession) GetLPPlusTotalSupply() (*big.Int, error) {
	return _InvariantsQuery.Contract.GetLPPlusTotalSupply(&_InvariantsQuery.CallOpts)
}

// GetLPPlusTotalSupply is a free data retrieval call binding the contract method 0x2a639cbe.
//
// Solidity: function getLPPlusTotalSupply() view returns(uint256)
func (_InvariantsQuery *InvariantsQueryCallerSession) GetLPPlusTotalSupply() (*big.Int, error) {
	return _InvariantsQuery.Contract.GetLPPlusTotalSupply(&_InvariantsQuery.CallOpts)
}

// GetPoolMetrics is a free data retrieval call binding the contract method 0x1866509f.
//
// Solidity: function getPoolMetrics() view returns((uint256,uint256,uint256,uint256))
func (_InvariantsQuery *InvariantsQueryCaller) GetPoolMetrics(opts *bind.CallOpts) (InvariantsQueryPoolMetrics, error) {
	var out []interface{}
	err := _InvariantsQuery.contract.Call(opts, &out, "getPoolMetrics")

	if err != nil {
		return *new(InvariantsQueryPoolMetrics), err
	}

	out0 := *abi.ConvertType(out[0], new(InvariantsQueryPoolMetrics)).(*InvariantsQueryPoolMetrics)

	return out0, err

}

// GetPoolMetrics is a free data retrieval call binding the contract method 0x1866509f.
//
// Solidity: function getPoolMetrics() view returns((uint256,uint256,uint256,uint256))
func (_InvariantsQuery *InvariantsQuerySession) GetPoolMetrics() (InvariantsQueryPoolMetrics, error) {
	return _InvariantsQuery.Contract.GetPoolMetrics(&_InvariantsQuery.CallOpts)
}

// GetPoolMetrics is a free data retrieval call binding the contract method 0x1866509f.
//
// Solidity: function getPoolMetrics() view returns((uint256,uint256,uint256,uint256))
func (_InvariantsQuery *InvariantsQueryCallerSession) GetPoolMetrics() (InvariantsQueryPoolMetrics, error) {
	return _InvariantsQuery.Contract.GetPoolMetrics(&_InvariantsQuery.CallOpts)
}

// GetSPPlusStates is a free data retrieval call binding the contract method 0xbfa57ecb.
//
// Solidity: function getSPPlusStates(uint256[] tokenIds) view returns((address,uint256)[] out)
func (_InvariantsQuery *InvariantsQueryCaller) GetSPPlusStates(opts *bind.CallOpts, tokenIds []*big.Int) ([]InvariantsQuerySPPlusTokenState, error) {
	var out []interface{}
	err := _InvariantsQuery.contract.Call(opts, &out, "getSPPlusStates", tokenIds)

	if err != nil {
		return *new([]InvariantsQuerySPPlusTokenState), err
	}

	out0 := *abi.ConvertType(out[0], new([]InvariantsQuerySPPlusTokenState)).(*[]InvariantsQuerySPPlusTokenState)

	return out0, err

}

// GetSPPlusStates is a free data retrieval call binding the contract method 0xbfa57ecb.
//
// Solidity: function getSPPlusStates(uint256[] tokenIds) view returns((address,uint256)[] out)
func (_InvariantsQuery *InvariantsQuerySession) GetSPPlusStates(tokenIds []*big.Int) ([]InvariantsQuerySPPlusTokenState, error) {
	return _InvariantsQuery.Contract.GetSPPlusStates(&_InvariantsQuery.CallOpts, tokenIds)
}

// GetSPPlusStates is a free data retrieval call binding the contract method 0xbfa57ecb.
//
// Solidity: function getSPPlusStates(uint256[] tokenIds) view returns((address,uint256)[] out)
func (_InvariantsQuery *InvariantsQueryCallerSession) GetSPPlusStates(tokenIds []*big.Int) ([]InvariantsQuerySPPlusTokenState, error) {
	return _InvariantsQuery.Contract.GetSPPlusStates(&_InvariantsQuery.CallOpts, tokenIds)
}

// GetSPPlusTokenIdsForAgents is a free data retrieval call binding the contract method 0xd99a7698.
//
// Solidity: function getSPPlusTokenIdsForAgents(uint256[] agentIds) view returns(uint256[] out)
func (_InvariantsQuery *InvariantsQueryCaller) GetSPPlusTokenIdsForAgents(opts *bind.CallOpts, agentIds []*big.Int) ([]*big.Int, error) {
	var out []interface{}
	err := _InvariantsQuery.contract.Call(opts, &out, "getSPPlusTokenIdsForAgents", agentIds)

	if err != nil {
		return *new([]*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new([]*big.Int)).(*[]*big.Int)

	return out0, err

}

// GetSPPlusTokenIdsForAgents is a free data retrieval call binding the contract method 0xd99a7698.
//
// Solidity: function getSPPlusTokenIdsForAgents(uint256[] agentIds) view returns(uint256[] out)
func (_InvariantsQuery *InvariantsQuerySession) GetSPPlusTokenIdsForAgents(agentIds []*big.Int) ([]*big.Int, error) {
	return _InvariantsQuery.Contract.GetSPPlusTokenIdsForAgents(&_InvariantsQuery.CallOpts, agentIds)
}

// GetSPPlusTokenIdsForAgents is a free data retrieval call binding the contract method 0xd99a7698.
//
// Solidity: function getSPPlusTokenIdsForAgents(uint256[] agentIds) view returns(uint256[] out)
func (_InvariantsQuery *InvariantsQueryCallerSession) GetSPPlusTokenIdsForAgents(agentIds []*big.Int) ([]*big.Int, error) {
	return _InvariantsQuery.Contract.GetSPPlusTokenIdsForAgents(&_InvariantsQuery.CallOpts, agentIds)
}

// Ifil is a free data retrieval call binding the contract method 0xdb6f5dcd.
//
// Solidity: function ifil() view returns(address)
func (_InvariantsQuery *InvariantsQueryCaller) Ifil(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _InvariantsQuery.contract.Call(opts, &out, "ifil")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// Ifil is a free data retrieval call binding the contract method 0xdb6f5dcd.
//
// Solidity: function ifil() view returns(address)
func (_InvariantsQuery *InvariantsQuerySession) Ifil() (common.Address, error) {
	return _InvariantsQuery.Contract.Ifil(&_InvariantsQuery.CallOpts)
}

// Ifil is a free data retrieval call binding the contract method 0xdb6f5dcd.
//
// Solidity: function ifil() view returns(address)
func (_InvariantsQuery *InvariantsQueryCallerSession) Ifil() (common.Address, error) {
	return _InvariantsQuery.Contract.Ifil(&_InvariantsQuery.CallOpts)
}

// LpPlus is a free data retrieval call binding the contract method 0x2fc92888.
//
// Solidity: function lpPlus() view returns(address)
func (_InvariantsQuery *InvariantsQueryCaller) LpPlus(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _InvariantsQuery.contract.Call(opts, &out, "lpPlus")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// LpPlus is a free data retrieval call binding the contract method 0x2fc92888.
//
// Solidity: function lpPlus() view returns(address)
func (_InvariantsQuery *InvariantsQuerySession) LpPlus() (common.Address, error) {
	return _InvariantsQuery.Contract.LpPlus(&_InvariantsQuery.CallOpts)
}

// LpPlus is a free data retrieval call binding the contract method 0x2fc92888.
//
// Solidity: function lpPlus() view returns(address)
func (_InvariantsQuery *InvariantsQueryCallerSession) LpPlus() (common.Address, error) {
	return _InvariantsQuery.Contract.LpPlus(&_InvariantsQuery.CallOpts)
}

// MinerRegistry is a free data retrieval call binding the contract method 0x90209f40.
//
// Solidity: function minerRegistry() view returns(address)
func (_InvariantsQuery *InvariantsQueryCaller) MinerRegistry(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _InvariantsQuery.contract.Call(opts, &out, "minerRegistry")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// MinerRegistry is a free data retrieval call binding the contract method 0x90209f40.
//
// Solidity: function minerRegistry() view returns(address)
func (_InvariantsQuery *InvariantsQuerySession) MinerRegistry() (common.Address, error) {
	return _InvariantsQuery.Contract.MinerRegistry(&_InvariantsQuery.CallOpts)
}

// MinerRegistry is a free data retrieval call binding the contract method 0x90209f40.
//
// Solidity: function minerRegistry() view returns(address)
func (_InvariantsQuery *InvariantsQueryCallerSession) MinerRegistry() (common.Address, error) {
	return _InvariantsQuery.Contract.MinerRegistry(&_InvariantsQuery.CallOpts)
}

// Pool is a free data retrieval call binding the contract method 0x16f0115b.
//
// Solidity: function pool() view returns(address)
func (_InvariantsQuery *InvariantsQueryCaller) Pool(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _InvariantsQuery.contract.Call(opts, &out, "pool")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// Pool is a free data retrieval call binding the contract method 0x16f0115b.
//
// Solidity: function pool() view returns(address)
func (_InvariantsQuery *InvariantsQuerySession) Pool() (common.Address, error) {
	return _InvariantsQuery.Contract.Pool(&_InvariantsQuery.CallOpts)
}

// Pool is a free data retrieval call binding the contract method 0x16f0115b.
//
// Solidity: function pool() view returns(address)
func (_InvariantsQuery *InvariantsQueryCallerSession) Pool() (common.Address, error) {
	return _InvariantsQuery.Contract.Pool(&_InvariantsQuery.CallOpts)
}

// Router is a free data retrieval call binding the contract method 0xf887ea40.
//
// Solidity: function router() view returns(address)
func (_InvariantsQuery *InvariantsQueryCaller) Router(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _InvariantsQuery.contract.Call(opts, &out, "router")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// Router is a free data retrieval call binding the contract method 0xf887ea40.
//
// Solidity: function router() view returns(address)
func (_InvariantsQuery *InvariantsQuerySession) Router() (common.Address, error) {
	return _InvariantsQuery.Contract.Router(&_InvariantsQuery.CallOpts)
}

// Router is a free data retrieval call binding the contract method 0xf887ea40.
//
// Solidity: function router() view returns(address)
func (_InvariantsQuery *InvariantsQueryCallerSession) Router() (common.Address, error) {
	return _InvariantsQuery.Contract.Router(&_InvariantsQuery.CallOpts)
}

// SpPlus is a free data retrieval call binding the contract method 0xa34ae1af.
//
// Solidity: function spPlus() view returns(address)
func (_InvariantsQuery *InvariantsQueryCaller) SpPlus(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _InvariantsQuery.contract.Call(opts, &out, "spPlus")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// SpPlus is a free data retrieval call binding the contract method 0xa34ae1af.
//
// Solidity: function spPlus() view returns(address)
func (_InvariantsQuery *InvariantsQuerySession) SpPlus() (common.Address, error) {
	return _InvariantsQuery.Contract.SpPlus(&_InvariantsQuery.CallOpts)
}

// SpPlus is a free data retrieval call binding the contract method 0xa34ae1af.
//
// Solidity: function spPlus() view returns(address)
func (_InvariantsQuery *InvariantsQueryCallerSession) SpPlus() (common.Address, error) {
	return _InvariantsQuery.Contract.SpPlus(&_InvariantsQuery.CallOpts)
}

// Code generated - DO NOT EDIT.
// This file is a generated binding and any manual changes will be lost.

package bindings

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

// DappLinkVRFFactoryMetaData contains all meta data concerning the DappLinkVRFFactory contract.
var DappLinkVRFFactoryMetaData = &bind.MetaData{
	ABI: "[{\"type\":\"function\",\"name\":\"createProxy\",\"inputs\":[{\"name\":\"implementation\",\"type\":\"address\",\"internalType\":\"address\"},{\"name\":\"dapplinkAddress\",\"type\":\"address\",\"internalType\":\"address\"}],\"outputs\":[{\"name\":\"\",\"type\":\"address\",\"internalType\":\"address\"}],\"stateMutability\":\"nonpayable\"},{\"type\":\"event\",\"name\":\"ProxyCreated\",\"inputs\":[{\"name\":\"mintProxyAddress\",\"type\":\"address\",\"indexed\":false,\"internalType\":\"address\"}],\"anonymous\":false},{\"type\":\"error\",\"name\":\"FailedDeployment\",\"inputs\":[]},{\"type\":\"error\",\"name\":\"InsufficientBalance\",\"inputs\":[{\"name\":\"balance\",\"type\":\"uint256\",\"internalType\":\"uint256\"},{\"name\":\"needed\",\"type\":\"uint256\",\"internalType\":\"uint256\"}]}]",
	Bin: "0x60808060405234601557610189908161001a8239f35b5f80fdfe60806040908082526004361015610014575f80fd5b5f3560e01c6325b5672714610027575f80fd5b34610140578160031936011261014057600435916001600160a01b03808416840361014057602435938185168095036101405780763d602d80600a3d3981f3363d3d373d3d3d363d7300000062ffffff6e5af43d82803e903d91602b57fd5bf39360881c16175f5260781b17602052603760095ff0169182156101445750813b156101405780519263485cc95560e01b845233600485015260248401525f8360448183865af1801561013657610109575b602092507efffc2da0b561cae30d9826d37709e9421c4725faebc226cbbb7ef5fc5e7349838251848152a151908152f35b67ffffffffffffffff83116101225760209281526100d8565b634e487b7160e01b5f52604160045260245ffd5b50513d5f823e3d90fd5b5f80fd5b63b06ebf3d60e01b8152600490fdfea26469706673582212202b56c8d6cd940fbb415d0206c1b83616d826ef07b46438a69181978eb0ab31fd64736f6c63430008190033",
}

// DappLinkVRFFactoryABI is the input ABI used to generate the binding from.
// Deprecated: Use DappLinkVRFFactoryMetaData.ABI instead.
var DappLinkVRFFactoryABI = DappLinkVRFFactoryMetaData.ABI

// DappLinkVRFFactoryBin is the compiled bytecode used for deploying new contracts.
// Deprecated: Use DappLinkVRFFactoryMetaData.Bin instead.
var DappLinkVRFFactoryBin = DappLinkVRFFactoryMetaData.Bin

// DeployDappLinkVRFFactory deploys a new Ethereum contract, binding an instance of DappLinkVRFFactory to it.
func DeployDappLinkVRFFactory(auth *bind.TransactOpts, backend bind.ContractBackend) (common.Address, *types.Transaction, *DappLinkVRFFactory, error) {
	parsed, err := DappLinkVRFFactoryMetaData.GetAbi()
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	if parsed == nil {
		return common.Address{}, nil, nil, errors.New("GetABI returned nil")
	}

	address, tx, contract, err := bind.DeployContract(auth, *parsed, common.FromHex(DappLinkVRFFactoryBin), backend)
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	return address, tx, &DappLinkVRFFactory{DappLinkVRFFactoryCaller: DappLinkVRFFactoryCaller{contract: contract}, DappLinkVRFFactoryTransactor: DappLinkVRFFactoryTransactor{contract: contract}, DappLinkVRFFactoryFilterer: DappLinkVRFFactoryFilterer{contract: contract}}, nil
}

// DappLinkVRFFactory is an auto generated Go binding around an Ethereum contract.
type DappLinkVRFFactory struct {
	DappLinkVRFFactoryCaller     // Read-only binding to the contract
	DappLinkVRFFactoryTransactor // Write-only binding to the contract
	DappLinkVRFFactoryFilterer   // Log filterer for contract events
}

// DappLinkVRFFactoryCaller is an auto generated read-only Go binding around an Ethereum contract.
type DappLinkVRFFactoryCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// DappLinkVRFFactoryTransactor is an auto generated write-only Go binding around an Ethereum contract.
type DappLinkVRFFactoryTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// DappLinkVRFFactoryFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type DappLinkVRFFactoryFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// DappLinkVRFFactorySession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type DappLinkVRFFactorySession struct {
	Contract     *DappLinkVRFFactory // Generic contract binding to set the session for
	CallOpts     bind.CallOpts       // Call options to use throughout this session
	TransactOpts bind.TransactOpts   // Transaction auth options to use throughout this session
}

// DappLinkVRFFactoryCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type DappLinkVRFFactoryCallerSession struct {
	Contract *DappLinkVRFFactoryCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts             // Call options to use throughout this session
}

// DappLinkVRFFactoryTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type DappLinkVRFFactoryTransactorSession struct {
	Contract     *DappLinkVRFFactoryTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts             // Transaction auth options to use throughout this session
}

// DappLinkVRFFactoryRaw is an auto generated low-level Go binding around an Ethereum contract.
type DappLinkVRFFactoryRaw struct {
	Contract *DappLinkVRFFactory // Generic contract binding to access the raw methods on
}

// DappLinkVRFFactoryCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type DappLinkVRFFactoryCallerRaw struct {
	Contract *DappLinkVRFFactoryCaller // Generic read-only contract binding to access the raw methods on
}

// DappLinkVRFFactoryTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type DappLinkVRFFactoryTransactorRaw struct {
	Contract *DappLinkVRFFactoryTransactor // Generic write-only contract binding to access the raw methods on
}

// NewDappLinkVRFFactory creates a new instance of DappLinkVRFFactory, bound to a specific deployed contract.
func NewDappLinkVRFFactory(address common.Address, backend bind.ContractBackend) (*DappLinkVRFFactory, error) {
	contract, err := bindDappLinkVRFFactory(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &DappLinkVRFFactory{DappLinkVRFFactoryCaller: DappLinkVRFFactoryCaller{contract: contract}, DappLinkVRFFactoryTransactor: DappLinkVRFFactoryTransactor{contract: contract}, DappLinkVRFFactoryFilterer: DappLinkVRFFactoryFilterer{contract: contract}}, nil
}

// NewDappLinkVRFFactoryCaller creates a new read-only instance of DappLinkVRFFactory, bound to a specific deployed contract.
func NewDappLinkVRFFactoryCaller(address common.Address, caller bind.ContractCaller) (*DappLinkVRFFactoryCaller, error) {
	contract, err := bindDappLinkVRFFactory(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &DappLinkVRFFactoryCaller{contract: contract}, nil
}

// NewDappLinkVRFFactoryTransactor creates a new write-only instance of DappLinkVRFFactory, bound to a specific deployed contract.
func NewDappLinkVRFFactoryTransactor(address common.Address, transactor bind.ContractTransactor) (*DappLinkVRFFactoryTransactor, error) {
	contract, err := bindDappLinkVRFFactory(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &DappLinkVRFFactoryTransactor{contract: contract}, nil
}

// NewDappLinkVRFFactoryFilterer creates a new log filterer instance of DappLinkVRFFactory, bound to a specific deployed contract.
func NewDappLinkVRFFactoryFilterer(address common.Address, filterer bind.ContractFilterer) (*DappLinkVRFFactoryFilterer, error) {
	contract, err := bindDappLinkVRFFactory(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &DappLinkVRFFactoryFilterer{contract: contract}, nil
}

// bindDappLinkVRFFactory binds a generic wrapper to an already deployed contract.
func bindDappLinkVRFFactory(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := DappLinkVRFFactoryMetaData.GetAbi()
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, *parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_DappLinkVRFFactory *DappLinkVRFFactoryRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _DappLinkVRFFactory.Contract.DappLinkVRFFactoryCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_DappLinkVRFFactory *DappLinkVRFFactoryRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _DappLinkVRFFactory.Contract.DappLinkVRFFactoryTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_DappLinkVRFFactory *DappLinkVRFFactoryRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _DappLinkVRFFactory.Contract.DappLinkVRFFactoryTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_DappLinkVRFFactory *DappLinkVRFFactoryCallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _DappLinkVRFFactory.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_DappLinkVRFFactory *DappLinkVRFFactoryTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _DappLinkVRFFactory.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_DappLinkVRFFactory *DappLinkVRFFactoryTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _DappLinkVRFFactory.Contract.contract.Transact(opts, method, params...)
}

// CreateProxy is a paid mutator transaction binding the contract method 0x25b56727.
//
// Solidity: function createProxy(address implementation, address dapplinkAddress) returns(address)
func (_DappLinkVRFFactory *DappLinkVRFFactoryTransactor) CreateProxy(opts *bind.TransactOpts, implementation common.Address, dapplinkAddress common.Address) (*types.Transaction, error) {
	return _DappLinkVRFFactory.contract.Transact(opts, "createProxy", implementation, dapplinkAddress)
}

// CreateProxy is a paid mutator transaction binding the contract method 0x25b56727.
//
// Solidity: function createProxy(address implementation, address dapplinkAddress) returns(address)
func (_DappLinkVRFFactory *DappLinkVRFFactorySession) CreateProxy(implementation common.Address, dapplinkAddress common.Address) (*types.Transaction, error) {
	return _DappLinkVRFFactory.Contract.CreateProxy(&_DappLinkVRFFactory.TransactOpts, implementation, dapplinkAddress)
}

// CreateProxy is a paid mutator transaction binding the contract method 0x25b56727.
//
// Solidity: function createProxy(address implementation, address dapplinkAddress) returns(address)
func (_DappLinkVRFFactory *DappLinkVRFFactoryTransactorSession) CreateProxy(implementation common.Address, dapplinkAddress common.Address) (*types.Transaction, error) {
	return _DappLinkVRFFactory.Contract.CreateProxy(&_DappLinkVRFFactory.TransactOpts, implementation, dapplinkAddress)
}

// DappLinkVRFFactoryProxyCreatedIterator is returned from FilterProxyCreated and is used to iterate over the raw logs and unpacked data for ProxyCreated events raised by the DappLinkVRFFactory contract.
type DappLinkVRFFactoryProxyCreatedIterator struct {
	Event *DappLinkVRFFactoryProxyCreated // Event containing the contract specifics and raw log

	contract *bind.BoundContract // Generic contract to use for unpacking event data
	event    string              // Event name to use for unpacking event data

	logs chan types.Log        // Log channel receiving the found contract events
	sub  ethereum.Subscription // Subscription for errors, completion and termination
	done bool                  // Whether the subscription completed delivering logs
	fail error                 // Occurred error to stop iteration
}

// Next advances the iterator to the subsequent event, returning whether there
// are any more events found. In case of a retrieval or parsing error, false is
// returned and Error() can be queried for the exact failure.
func (it *DappLinkVRFFactoryProxyCreatedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(DappLinkVRFFactoryProxyCreated)
			if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
				it.fail = err
				return false
			}
			it.Event.Raw = log
			return true

		default:
			return false
		}
	}
	// Iterator still in progress, wait for either a data or an error event
	select {
	case log := <-it.logs:
		it.Event = new(DappLinkVRFFactoryProxyCreated)
		if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
			it.fail = err
			return false
		}
		it.Event.Raw = log
		return true

	case err := <-it.sub.Err():
		it.done = true
		it.fail = err
		return it.Next()
	}
}

// Error returns any retrieval or parsing error occurred during filtering.
func (it *DappLinkVRFFactoryProxyCreatedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *DappLinkVRFFactoryProxyCreatedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// DappLinkVRFFactoryProxyCreated represents a ProxyCreated event raised by the DappLinkVRFFactory contract.
type DappLinkVRFFactoryProxyCreated struct {
	MintProxyAddress common.Address
	Raw              types.Log // Blockchain specific contextual infos
}

// FilterProxyCreated is a free log retrieval operation binding the contract event 0x00fffc2da0b561cae30d9826d37709e9421c4725faebc226cbbb7ef5fc5e7349.
//
// Solidity: event ProxyCreated(address mintProxyAddress)
func (_DappLinkVRFFactory *DappLinkVRFFactoryFilterer) FilterProxyCreated(opts *bind.FilterOpts) (*DappLinkVRFFactoryProxyCreatedIterator, error) {

	logs, sub, err := _DappLinkVRFFactory.contract.FilterLogs(opts, "ProxyCreated")
	if err != nil {
		return nil, err
	}
	return &DappLinkVRFFactoryProxyCreatedIterator{contract: _DappLinkVRFFactory.contract, event: "ProxyCreated", logs: logs, sub: sub}, nil
}

// WatchProxyCreated is a free log subscription operation binding the contract event 0x00fffc2da0b561cae30d9826d37709e9421c4725faebc226cbbb7ef5fc5e7349.
//
// Solidity: event ProxyCreated(address mintProxyAddress)
func (_DappLinkVRFFactory *DappLinkVRFFactoryFilterer) WatchProxyCreated(opts *bind.WatchOpts, sink chan<- *DappLinkVRFFactoryProxyCreated) (event.Subscription, error) {

	logs, sub, err := _DappLinkVRFFactory.contract.WatchLogs(opts, "ProxyCreated")
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(DappLinkVRFFactoryProxyCreated)
				if err := _DappLinkVRFFactory.contract.UnpackLog(event, "ProxyCreated", log); err != nil {
					return err
				}
				event.Raw = log

				select {
				case sink <- event:
				case err := <-sub.Err():
					return err
				case <-quit:
					return nil
				}
			case err := <-sub.Err():
				return err
			case <-quit:
				return nil
			}
		}
	}), nil
}

// ParseProxyCreated is a log parse operation binding the contract event 0x00fffc2da0b561cae30d9826d37709e9421c4725faebc226cbbb7ef5fc5e7349.
//
// Solidity: event ProxyCreated(address mintProxyAddress)
func (_DappLinkVRFFactory *DappLinkVRFFactoryFilterer) ParseProxyCreated(log types.Log) (*DappLinkVRFFactoryProxyCreated, error) {
	event := new(DappLinkVRFFactoryProxyCreated)
	if err := _DappLinkVRFFactory.contract.UnpackLog(event, "ProxyCreated", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

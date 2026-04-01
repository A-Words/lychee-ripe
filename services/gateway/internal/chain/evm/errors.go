package evm

import "errors"

var (
	ErrNodeUnavailable = errors.New("evm node unavailable")
	ErrTxReverted      = errors.New("evm transaction reverted")
	ErrContractCall    = errors.New("evm contract call failed")
	ErrAnchorNotFound  = errors.New("evm anchor not found")
	ErrInvalidConfig   = errors.New("evm invalid config")
	ErrInvalidInput    = errors.New("evm invalid input")
)

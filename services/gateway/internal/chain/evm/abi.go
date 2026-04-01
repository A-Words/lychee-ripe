package evm

import (
	"strings"

	"github.com/ethereum/go-ethereum/accounts/abi"
)

const contractABIJSON = `[
  {
    "inputs": [
      { "internalType": "string", "name": "batchId", "type": "string" },
      { "internalType": "bytes32", "name": "anchorHash", "type": "bytes32" },
      { "internalType": "uint256", "name": "timestamp", "type": "uint256" }
    ],
    "name": "anchorBatch",
    "outputs": [],
    "stateMutability": "nonpayable",
    "type": "function"
  },
  {
    "inputs": [
      { "internalType": "string", "name": "batchId", "type": "string" }
    ],
    "name": "getBatchAnchor",
    "outputs": [
      { "internalType": "bytes32", "name": "anchorHash", "type": "bytes32" },
      { "internalType": "uint256", "name": "anchoredAt", "type": "uint256" },
      { "internalType": "bool", "name": "exists", "type": "bool" }
    ],
    "stateMutability": "view",
    "type": "function"
  }
]`

func parseContractABI() (abi.ABI, error) {
	return abi.JSON(strings.NewReader(contractABIJSON))
}

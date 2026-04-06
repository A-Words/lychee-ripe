package evm

import (
	"context"
	"crypto/ecdsa"
	"encoding/hex"
	"errors"
	"fmt"
	"math/big"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/lychee-ripe/gateway/internal/config"
	"github.com/lychee-ripe/gateway/internal/domain"
)

type Adapter struct {
	client              rpcClient
	chainID             *big.Int
	contractAddress     common.Address
	privateKey          *ecdsa.PrivateKey
	fromAddress         common.Address
	contractABI         anyABI
	txTimeout           time.Duration
	receiptPollInterval time.Duration
}

type anyABI interface {
	Pack(name string, args ...any) ([]byte, error)
	Unpack(name string, data []byte) ([]any, error)
}

func NewAdapter(ctx context.Context, cfg config.ChainConfig) (*Adapter, error) {
	client, err := dialRPC(strings.TrimSpace(cfg.RPCURL))
	if err != nil {
		return nil, wrapNodeError("dial rpc", err)
	}

	adapter, err := newAdapterWithClient(ctx, cfg, client)
	if err != nil {
		client.Close()
		return nil, err
	}
	return adapter, nil
}

func newAdapterWithClient(ctx context.Context, cfg config.ChainConfig, client rpcClient) (*Adapter, error) {
	chainID, err := parseChainID(cfg.ChainID)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidConfig, err)
	}

	if !common.IsHexAddress(strings.TrimSpace(cfg.ContractAddress)) {
		return nil, fmt.Errorf("%w: invalid contract_address", ErrInvalidConfig)
	}

	privateKey, fromAddr, err := parsePrivateKey(cfg.PrivateKey)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidConfig, err)
	}

	parsedABI, err := parseContractABI()
	if err != nil {
		return nil, fmt.Errorf("%w: parse contract abi: %v", ErrInvalidConfig, err)
	}

	nodeChainID, err := client.ChainID(ctx)
	if err != nil {
		return nil, wrapNodeError("query chain id", err)
	}
	if nodeChainID.Cmp(chainID) != 0 {
		return nil, fmt.Errorf("%w: configured chain_id=%s does not match node chain_id=%s", ErrInvalidConfig, chainID.String(), nodeChainID.String())
	}

	txTimeout := time.Duration(cfg.TxTimeoutS) * time.Second
	if txTimeout <= 0 {
		txTimeout = 30 * time.Second
	}
	pollInterval := time.Duration(cfg.ReceiptPollIntervalMS) * time.Millisecond
	if pollInterval <= 0 {
		pollInterval = 500 * time.Millisecond
	}

	return &Adapter{
		client:              client,
		chainID:             chainID,
		contractAddress:     common.HexToAddress(strings.TrimSpace(cfg.ContractAddress)),
		privateKey:          privateKey,
		fromAddress:         fromAddr,
		contractABI:         parsedABI,
		txTimeout:           txTimeout,
		receiptPollInterval: pollInterval,
	}, nil
}

func (a *Adapter) Close() {
	if a == nil || a.client == nil {
		return
	}
	a.client.Close()
}

func (a *Adapter) AnchorBatch(ctx context.Context, req AnchorBatchRequest) (domain.AnchorProofRecord, error) {
	if strings.TrimSpace(req.BatchID) == "" {
		return domain.AnchorProofRecord{}, fmt.Errorf("%w: batch_id is required", ErrInvalidInput)
	}
	hashBytes, normalizedHash, err := parseAnchorHash(req.AnchorHash)
	if err != nil {
		return domain.AnchorProofRecord{}, err
	}

	anchoredAt := req.Timestamp.UTC()
	if req.Timestamp.IsZero() {
		anchoredAt = time.Now().UTC()
	}
	timestamp := big.NewInt(anchoredAt.Unix())

	callData, err := a.contractABI.Pack("anchorBatch", strings.TrimSpace(req.BatchID), hashBytes, timestamp)
	if err != nil {
		return domain.AnchorProofRecord{}, fmt.Errorf("%w: pack anchorBatch args: %v", ErrContractCall, err)
	}

	txCtx, cancel := context.WithTimeout(ctx, a.txTimeout)
	defer cancel()

	nonce, err := a.client.PendingNonceAt(txCtx, a.fromAddress)
	if err != nil {
		return domain.AnchorProofRecord{}, wrapNodeError("get pending nonce", err)
	}

	gasPrice, err := a.client.SuggestGasPrice(txCtx)
	if err != nil {
		return domain.AnchorProofRecord{}, wrapNodeError("suggest gas price", err)
	}

	gasLimit, err := a.client.EstimateGas(txCtx, ethereum.CallMsg{
		From: a.fromAddress,
		To:   &a.contractAddress,
		Data: callData,
	})
	if err != nil {
		if isNodeUnavailableError(err) {
			return domain.AnchorProofRecord{}, wrapNodeError("estimate gas", err)
		}
		return domain.AnchorProofRecord{}, fmt.Errorf("%w: estimate gas: %v", ErrContractCall, err)
	}

	unsignedTx := types.NewTransaction(
		nonce,
		a.contractAddress,
		big.NewInt(0),
		gasLimit,
		gasPrice,
		callData,
	)

	signedTx, err := types.SignTx(unsignedTx, types.NewEIP155Signer(a.chainID), a.privateKey)
	if err != nil {
		return domain.AnchorProofRecord{}, fmt.Errorf("%w: sign transaction: %v", ErrContractCall, err)
	}

	if err := a.client.SendTransaction(txCtx, signedTx); err != nil {
		if isNodeUnavailableError(err) {
			return domain.AnchorProofRecord{}, wrapNodeError("send transaction", err)
		}
		return domain.AnchorProofRecord{}, fmt.Errorf("%w: send transaction: %v", ErrContractCall, err)
	}

	receipt, err := a.waitReceipt(txCtx, signedTx.Hash())
	if err != nil {
		return domain.AnchorProofRecord{}, err
	}
	if receipt.Status == types.ReceiptStatusFailed {
		return domain.AnchorProofRecord{}, fmt.Errorf("%w: tx_hash=%s", ErrTxReverted, signedTx.Hash().Hex())
	}
	if receipt.BlockNumber == nil {
		return domain.AnchorProofRecord{}, fmt.Errorf("%w: missing block number in receipt", ErrContractCall)
	}
	if !receipt.BlockNumber.IsInt64() {
		return domain.AnchorProofRecord{}, fmt.Errorf("%w: invalid block number in receipt", ErrContractCall)
	}

	return domain.AnchorProofRecord{
		TxHash:          signedTx.Hash().Hex(),
		BlockNumber:     receipt.BlockNumber.Int64(),
		ChainID:         a.chainID.String(),
		ContractAddress: a.contractAddress.Hex(),
		AnchorHash:      normalizedHash,
		AnchoredAt:      anchoredAt,
	}, nil
}

func (a *Adapter) GetBatchAnchor(ctx context.Context, batchID string) (BatchAnchorOnChain, error) {
	if strings.TrimSpace(batchID) == "" {
		return BatchAnchorOnChain{}, fmt.Errorf("%w: batch_id is required", ErrInvalidInput)
	}

	callData, err := a.contractABI.Pack("getBatchAnchor", strings.TrimSpace(batchID))
	if err != nil {
		return BatchAnchorOnChain{}, fmt.Errorf("%w: pack getBatchAnchor args: %v", ErrContractCall, err)
	}

	callCtx, cancel := context.WithTimeout(ctx, a.txTimeout)
	defer cancel()

	raw, err := a.client.CallContract(callCtx, ethereum.CallMsg{
		To:   &a.contractAddress,
		Data: callData,
	}, nil)
	if err != nil {
		if isNodeUnavailableError(err) {
			return BatchAnchorOnChain{}, wrapNodeError("call getBatchAnchor", err)
		}
		return BatchAnchorOnChain{}, fmt.Errorf("%w: call getBatchAnchor: %v", ErrContractCall, err)
	}

	outputs, err := a.contractABI.Unpack("getBatchAnchor", raw)
	if err != nil {
		return BatchAnchorOnChain{}, fmt.Errorf("%w: unpack getBatchAnchor result: %v", ErrContractCall, err)
	}
	if len(outputs) != 3 {
		return BatchAnchorOnChain{}, fmt.Errorf("%w: unexpected getBatchAnchor outputs", ErrContractCall)
	}

	hashBytes, ok := outputs[0].([32]byte)
	if !ok {
		return BatchAnchorOnChain{}, fmt.Errorf("%w: invalid anchor_hash output type", ErrContractCall)
	}
	anchoredAtValue, ok := outputs[1].(*big.Int)
	if !ok {
		return BatchAnchorOnChain{}, fmt.Errorf("%w: invalid anchored_at output type", ErrContractCall)
	}
	exists, ok := outputs[2].(bool)
	if !ok {
		return BatchAnchorOnChain{}, fmt.Errorf("%w: invalid exists output type", ErrContractCall)
	}
	if !exists {
		return BatchAnchorOnChain{}, fmt.Errorf("%w: batch_id=%s", ErrAnchorNotFound, strings.TrimSpace(batchID))
	}
	if !anchoredAtValue.IsInt64() {
		return BatchAnchorOnChain{}, fmt.Errorf("%w: anchored_at exceeds int64", ErrContractCall)
	}

	return BatchAnchorOnChain{
		BatchID:    strings.TrimSpace(batchID),
		AnchorHash: "0x" + hex.EncodeToString(hashBytes[:]),
		AnchoredAt: time.Unix(anchoredAtValue.Int64(), 0).UTC(),
	}, nil
}

func (a *Adapter) waitReceipt(ctx context.Context, txHash common.Hash) (*types.Receipt, error) {
	ticker := time.NewTicker(a.receiptPollInterval)
	defer ticker.Stop()

	for {
		receipt, err := a.client.TransactionReceipt(ctx, txHash)
		if err == nil {
			return receipt, nil
		}

		if errors.Is(err, ethereum.NotFound) {
			select {
			case <-ctx.Done():
				return nil, wrapNodeError("wait receipt timeout", ctx.Err())
			case <-ticker.C:
				continue
			}
		}

		if isNodeUnavailableError(err) {
			return nil, wrapNodeError("query tx receipt", err)
		}
		return nil, fmt.Errorf("%w: query tx receipt: %v", ErrContractCall, err)
	}
}

func parseChainID(raw string) (*big.Int, error) {
	clean := strings.TrimSpace(raw)
	if clean == "" {
		return nil, errors.New("chain_id is required")
	}
	value, ok := new(big.Int).SetString(clean, 0)
	if !ok || value.Sign() <= 0 {
		return nil, errors.New("chain_id must be a positive integer")
	}
	if !value.IsInt64() {
		return nil, errors.New("chain_id exceeds int64 range")
	}
	return value, nil
}

func parsePrivateKey(raw string) (*ecdsa.PrivateKey, common.Address, error) {
	clean := strings.TrimPrefix(strings.TrimSpace(raw), "0x")
	privateKey, err := crypto.HexToECDSA(clean)
	if err != nil {
		return nil, common.Address{}, errors.New("private_key must be a valid 32-byte hex key")
	}
	from := crypto.PubkeyToAddress(privateKey.PublicKey)
	return privateKey, from, nil
}

func parseAnchorHash(raw string) ([32]byte, string, error) {
	var out [32]byte

	clean := strings.TrimPrefix(strings.TrimSpace(raw), "0x")
	if len(clean) != 64 {
		return out, "", fmt.Errorf("%w: anchor_hash must be 32-byte hex string", ErrInvalidInput)
	}

	decoded, err := hex.DecodeString(clean)
	if err != nil {
		return out, "", fmt.Errorf("%w: anchor_hash must be hex encoded", ErrInvalidInput)
	}
	if len(decoded) != len(out) {
		return out, "", fmt.Errorf("%w: anchor_hash length mismatch", ErrInvalidInput)
	}
	copy(out[:], decoded)
	return out, "0x" + strings.ToLower(clean), nil
}

func wrapNodeError(action string, err error) error {
	return fmt.Errorf("%w: %s: %v", ErrNodeUnavailable, action, err)
}

func isNodeUnavailableError(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return true
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "connection refused") ||
		strings.Contains(msg, "no such host") ||
		strings.Contains(msg, "i/o timeout") ||
		strings.Contains(msg, "context deadline exceeded") ||
		strings.Contains(msg, "eof") ||
		strings.Contains(msg, "connection reset by peer")
}

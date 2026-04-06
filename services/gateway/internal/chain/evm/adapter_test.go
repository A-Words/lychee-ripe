package evm

import (
	"context"
	"errors"
	"math/big"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/lychee-ripe/gateway/internal/config"
)

type fakeRPCClient struct {
	chainIDFn            func(context.Context) (*big.Int, error)
	pendingNonceAtFn     func(context.Context, common.Address) (uint64, error)
	suggestGasPriceFn    func(context.Context) (*big.Int, error)
	estimateGasFn        func(context.Context, ethereum.CallMsg) (uint64, error)
	sendTransactionFn    func(context.Context, *types.Transaction) error
	transactionReceiptFn func(context.Context, common.Hash) (*types.Receipt, error)
	callContractFn       func(context.Context, ethereum.CallMsg, *big.Int) ([]byte, error)
}

func (f *fakeRPCClient) ChainID(ctx context.Context) (*big.Int, error) {
	if f.chainIDFn != nil {
		return f.chainIDFn(ctx)
	}
	return big.NewInt(31337), nil
}

func (f *fakeRPCClient) PendingNonceAt(ctx context.Context, account common.Address) (uint64, error) {
	if f.pendingNonceAtFn != nil {
		return f.pendingNonceAtFn(ctx, account)
	}
	return 1, nil
}

func (f *fakeRPCClient) SuggestGasPrice(ctx context.Context) (*big.Int, error) {
	if f.suggestGasPriceFn != nil {
		return f.suggestGasPriceFn(ctx)
	}
	return big.NewInt(1_000_000_000), nil
}

func (f *fakeRPCClient) EstimateGas(ctx context.Context, msg ethereum.CallMsg) (uint64, error) {
	if f.estimateGasFn != nil {
		return f.estimateGasFn(ctx, msg)
	}
	return 210000, nil
}

func (f *fakeRPCClient) SendTransaction(ctx context.Context, tx *types.Transaction) error {
	if f.sendTransactionFn != nil {
		return f.sendTransactionFn(ctx, tx)
	}
	return nil
}

func (f *fakeRPCClient) TransactionReceipt(ctx context.Context, txHash common.Hash) (*types.Receipt, error) {
	if f.transactionReceiptFn != nil {
		return f.transactionReceiptFn(ctx, txHash)
	}
	return nil, ethereum.NotFound
}

func (f *fakeRPCClient) CallContract(ctx context.Context, msg ethereum.CallMsg, blockNumber *big.Int) ([]byte, error) {
	if f.callContractFn != nil {
		return f.callContractFn(ctx, msg, blockNumber)
	}
	return nil, nil
}

func (f *fakeRPCClient) Close() {}

func TestAnchorBatchSuccess(t *testing.T) {
	t.Parallel()

	var sentTxHash common.Hash
	receiptCall := 0
	client := &fakeRPCClient{
		sendTransactionFn: func(_ context.Context, tx *types.Transaction) error {
			sentTxHash = tx.Hash()
			return nil
		},
		transactionReceiptFn: func(_ context.Context, txHash common.Hash) (*types.Receipt, error) {
			receiptCall++
			if receiptCall == 1 {
				return nil, ethereum.NotFound
			}
			return &types.Receipt{
				Status:      types.ReceiptStatusSuccessful,
				BlockNumber: big.NewInt(101),
				TxHash:      txHash,
			}, nil
		},
	}

	adapter := mustNewTestAdapter(t, client)
	defer adapter.Close()

	req := AnchorBatchRequest{
		BatchID:    "batch-001",
		AnchorHash: "0x1111111111111111111111111111111111111111111111111111111111111111",
		Timestamp:  time.Unix(1_777_777_777, 0).UTC(),
	}

	proof, err := adapter.AnchorBatch(context.Background(), req)
	if err != nil {
		t.Fatalf("AnchorBatch failed: %v", err)
	}
	if proof.TxHash == "" || proof.TxHash != sentTxHash.Hex() {
		t.Fatalf("tx_hash = %q, want %q", proof.TxHash, sentTxHash.Hex())
	}
	if proof.BlockNumber != 101 {
		t.Fatalf("block_number = %d, want 101", proof.BlockNumber)
	}
	if proof.ChainID != "31337" {
		t.Fatalf("chain_id = %q, want 31337", proof.ChainID)
	}
	if proof.ContractAddress != "0x1234567890AbcdEF1234567890aBcdef12345678" {
		t.Fatalf("contract_address = %q", proof.ContractAddress)
	}
	if proof.AnchorHash != "0x1111111111111111111111111111111111111111111111111111111111111111" {
		t.Fatalf("anchor_hash = %q", proof.AnchorHash)
	}
}

func TestAnchorBatchNodeUnavailable(t *testing.T) {
	t.Parallel()

	client := &fakeRPCClient{
		suggestGasPriceFn: func(context.Context) (*big.Int, error) {
			return nil, errors.New("dial tcp 127.0.0.1:8545: connect: connection refused")
		},
	}
	adapter := mustNewTestAdapter(t, client)
	defer adapter.Close()

	_, err := adapter.AnchorBatch(context.Background(), AnchorBatchRequest{
		BatchID:    "batch-002",
		AnchorHash: "0x2222222222222222222222222222222222222222222222222222222222222222",
		Timestamp:  time.Now().UTC(),
	})
	if !errors.Is(err, ErrNodeUnavailable) {
		t.Fatalf("error = %v, want ErrNodeUnavailable", err)
	}
}

func TestAnchorBatchTxReverted(t *testing.T) {
	t.Parallel()

	client := &fakeRPCClient{
		transactionReceiptFn: func(_ context.Context, txHash common.Hash) (*types.Receipt, error) {
			return &types.Receipt{
				Status:      types.ReceiptStatusFailed,
				BlockNumber: big.NewInt(202),
				TxHash:      txHash,
			}, nil
		},
	}
	adapter := mustNewTestAdapter(t, client)
	defer adapter.Close()

	_, err := adapter.AnchorBatch(context.Background(), AnchorBatchRequest{
		BatchID:    "batch-003",
		AnchorHash: "0x3333333333333333333333333333333333333333333333333333333333333333",
		Timestamp:  time.Now().UTC(),
	})
	if !errors.Is(err, ErrTxReverted) {
		t.Fatalf("error = %v, want ErrTxReverted", err)
	}
}

func TestGetBatchAnchorSuccess(t *testing.T) {
	t.Parallel()

	parsedABI, err := parseContractABI()
	if err != nil {
		t.Fatalf("parse abi: %v", err)
	}
	hashBytes, _, err := parseAnchorHash("0xaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa")
	if err != nil {
		t.Fatalf("parse hash: %v", err)
	}
	rawOutput, err := parsedABI.Methods["getBatchAnchor"].Outputs.Pack(hashBytes, big.NewInt(1_777_777_888), true)
	if err != nil {
		t.Fatalf("pack output: %v", err)
	}

	client := &fakeRPCClient{
		callContractFn: func(context.Context, ethereum.CallMsg, *big.Int) ([]byte, error) {
			return rawOutput, nil
		},
	}
	adapter := mustNewTestAdapter(t, client)
	defer adapter.Close()

	record, err := adapter.GetBatchAnchor(context.Background(), "batch-004")
	if err != nil {
		t.Fatalf("GetBatchAnchor failed: %v", err)
	}
	if record.BatchID != "batch-004" {
		t.Fatalf("batch_id = %q, want batch-004", record.BatchID)
	}
	if record.AnchorHash != "0xaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa" {
		t.Fatalf("anchor_hash = %q", record.AnchorHash)
	}
	if record.AnchoredAt.Unix() != 1_777_777_888 {
		t.Fatalf("anchored_at unix = %d, want 1777777888", record.AnchoredAt.Unix())
	}
}

func TestGetBatchAnchorNotFound(t *testing.T) {
	t.Parallel()

	parsedABI, err := parseContractABI()
	if err != nil {
		t.Fatalf("parse abi: %v", err)
	}
	hashBytes, _, err := parseAnchorHash("0xbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb")
	if err != nil {
		t.Fatalf("parse hash: %v", err)
	}
	rawOutput, err := parsedABI.Methods["getBatchAnchor"].Outputs.Pack(hashBytes, big.NewInt(0), false)
	if err != nil {
		t.Fatalf("pack output: %v", err)
	}

	client := &fakeRPCClient{
		callContractFn: func(context.Context, ethereum.CallMsg, *big.Int) ([]byte, error) {
			return rawOutput, nil
		},
	}
	adapter := mustNewTestAdapter(t, client)
	defer adapter.Close()

	_, err = adapter.GetBatchAnchor(context.Background(), "batch-005")
	if !errors.Is(err, ErrAnchorNotFound) {
		t.Fatalf("error = %v, want ErrAnchorNotFound", err)
	}
}

func TestAnchorBatchInvalidInput(t *testing.T) {
	t.Parallel()

	adapter := mustNewTestAdapter(t, &fakeRPCClient{})
	defer adapter.Close()

	_, err := adapter.AnchorBatch(context.Background(), AnchorBatchRequest{
		BatchID:    "batch-006",
		AnchorHash: "0x1234",
	})
	if !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("error = %v, want ErrInvalidInput", err)
	}
}

func mustNewTestAdapter(t *testing.T, client rpcClient) *Adapter {
	t.Helper()
	cfg := config.ChainConfig{
		RPCURL:                "http://127.0.0.1:8545",
		ChainID:               "31337",
		ContractAddress:       "0x1234567890abcdef1234567890abcdef12345678",
		PrivateKey:            "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef",
		TxTimeoutS:            3,
		ReceiptPollIntervalMS: 1,
	}
	adapter, err := newAdapterWithClient(context.Background(), cfg, client)
	if err != nil {
		t.Fatalf("newAdapterWithClient failed: %v", err)
	}
	return adapter
}

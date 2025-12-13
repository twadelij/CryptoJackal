package wallet

import (
	"context"
	"crypto/ecdsa"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"go.uber.org/zap"
)

// Wallet manages Ethereum wallet operations
type Wallet struct {
	client     *ethclient.Client
	privateKey *ecdsa.PrivateKey
	address    common.Address
	chainID    *big.Int
	logger     *zap.Logger
}

// New creates a new wallet instance
func New(nodeURL string, privateKeyHex string, chainID int64, logger *zap.Logger) (*Wallet, error) {
	client, err := ethclient.Dial(nodeURL)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to Ethereum node: %w", err)
	}

	var privateKey *ecdsa.PrivateKey
	var address common.Address

	if privateKeyHex != "" {
		privateKey, err = crypto.HexToECDSA(privateKeyHex)
		if err != nil {
			return nil, fmt.Errorf("invalid private key: %w", err)
		}
		publicKey := privateKey.Public()
		publicKeyECDSA, ok := publicKey.(*ecdsa.PublicKey)
		if !ok {
			return nil, fmt.Errorf("failed to get public key")
		}
		address = crypto.PubkeyToAddress(*publicKeyECDSA)
	}

	return &Wallet{
		client:     client,
		privateKey: privateKey,
		address:    address,
		chainID:    big.NewInt(chainID),
		logger:     logger,
	}, nil
}

// Address returns the wallet address
func (w *Wallet) Address() common.Address {
	return w.address
}

// GetBalance returns the ETH balance in wei
func (w *Wallet) GetBalance(ctx context.Context) (*big.Int, error) {
	return w.client.BalanceAt(ctx, w.address, nil)
}

// GetBalanceETH returns the ETH balance as a float
func (w *Wallet) GetBalanceETH(ctx context.Context) (float64, error) {
	balance, err := w.GetBalance(ctx)
	if err != nil {
		return 0, err
	}
	// Convert wei to ETH
	fBalance := new(big.Float).SetInt(balance)
	ethValue := new(big.Float).Quo(fBalance, big.NewFloat(1e18))
	result, _ := ethValue.Float64()
	return result, nil
}

// GetNonce returns the next nonce for the wallet
func (w *Wallet) GetNonce(ctx context.Context) (uint64, error) {
	return w.client.PendingNonceAt(ctx, w.address)
}

// GetGasPrice returns the current gas price
func (w *Wallet) GetGasPrice(ctx context.Context) (*big.Int, error) {
	return w.client.SuggestGasPrice(ctx)
}

// SignTransaction signs a transaction
func (w *Wallet) SignTransaction(tx *types.Transaction) (*types.Transaction, error) {
	if w.privateKey == nil {
		return nil, fmt.Errorf("no private key configured")
	}
	return types.SignTx(tx, types.NewEIP155Signer(w.chainID), w.privateKey)
}

// SendTransaction sends a signed transaction
func (w *Wallet) SendTransaction(ctx context.Context, tx *types.Transaction) error {
	return w.client.SendTransaction(ctx, tx)
}

// Client returns the underlying eth client
func (w *Wallet) Client() *ethclient.Client {
	return w.client
}

// IsConfigured returns true if the wallet has a private key
func (w *Wallet) IsConfigured() bool {
	return w.privateKey != nil
}

// Close closes the wallet connection
func (w *Wallet) Close() {
	if w.client != nil {
		w.client.Close()
	}
}

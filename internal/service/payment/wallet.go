package payment

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/seu-repo/sigec-ve/internal/domain"
	"github.com/seu-repo/sigec-ve/internal/ports"
)

// WalletService implements ports.WalletService
type WalletService struct {
	repo ports.WalletRepository
	log  *zap.Logger
}

// NewWalletService creates a new wallet service
func NewWalletService(repo ports.WalletRepository, log *zap.Logger) *WalletService {
	return &WalletService{
		repo: repo,
		log:  log,
	}
}

// GetWallet retrieves or creates a user's wallet
func (s *WalletService) GetWallet(ctx context.Context, userID string) (*domain.Wallet, error) {
	wallet, err := s.repo.GetByUserID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get wallet: %w", err)
	}

	// Create wallet if it doesn't exist
	if wallet == nil {
		wallet = &domain.Wallet{
			ID:        uuid.New().String(),
			UserID:    userID,
			Balance:   0,
			Currency:  "BRL",
			UpdatedAt: time.Now(),
		}

		if err := s.repo.Save(ctx, wallet); err != nil {
			return nil, fmt.Errorf("failed to create wallet: %w", err)
		}

		s.log.Info("Created new wallet",
			zap.String("user_id", userID),
			zap.String("wallet_id", wallet.ID),
		)
	}

	return wallet, nil
}

// AddFunds adds funds to the wallet
func (s *WalletService) AddFunds(ctx context.Context, userID string, amount float64, paymentID string) error {
	if amount <= 0 {
		return fmt.Errorf("amount must be positive")
	}

	wallet, err := s.GetWallet(ctx, userID)
	if err != nil {
		return err
	}

	// Update balance
	newBalance := wallet.Balance + amount
	wallet.Balance = newBalance
	wallet.UpdatedAt = time.Now()

	if err := s.repo.Save(ctx, wallet); err != nil {
		return fmt.Errorf("failed to update wallet balance: %w", err)
	}

	// Record transaction
	tx := &domain.WalletTransaction{
		ID:          uuid.New().String(),
		WalletID:    wallet.ID,
		UserID:      userID,
		Type:        "credit",
		Amount:      amount,
		Balance:     newBalance,
		Description: "Funds added to wallet",
		ReferenceID: paymentID,
		CreatedAt:   time.Now(),
	}

	if err := s.repo.SaveTransaction(ctx, tx); err != nil {
		s.log.Error("Failed to save wallet transaction",
			zap.String("wallet_id", wallet.ID),
			zap.Error(err),
		)
	}

	s.log.Info("Funds added to wallet",
		zap.String("user_id", userID),
		zap.Float64("amount", amount),
		zap.Float64("new_balance", newBalance),
	)

	return nil
}

// DeductFunds deducts funds from the wallet
func (s *WalletService) DeductFunds(ctx context.Context, userID string, amount float64, description string, referenceID string) error {
	if amount <= 0 {
		return fmt.Errorf("amount must be positive")
	}

	wallet, err := s.GetWallet(ctx, userID)
	if err != nil {
		return err
	}

	if wallet.Balance < amount {
		return fmt.Errorf("insufficient balance: have %.2f, need %.2f", wallet.Balance, amount)
	}

	// Update balance
	newBalance := wallet.Balance - amount
	wallet.Balance = newBalance
	wallet.UpdatedAt = time.Now()

	if err := s.repo.Save(ctx, wallet); err != nil {
		return fmt.Errorf("failed to update wallet balance: %w", err)
	}

	// Record transaction
	tx := &domain.WalletTransaction{
		ID:          uuid.New().String(),
		WalletID:    wallet.ID,
		UserID:      userID,
		Type:        "debit",
		Amount:      amount,
		Balance:     newBalance,
		Description: description,
		ReferenceID: referenceID,
		CreatedAt:   time.Now(),
	}

	if err := s.repo.SaveTransaction(ctx, tx); err != nil {
		s.log.Error("Failed to save wallet transaction",
			zap.String("wallet_id", wallet.ID),
			zap.Error(err),
		)
	}

	s.log.Info("Funds deducted from wallet",
		zap.String("user_id", userID),
		zap.Float64("amount", amount),
		zap.Float64("new_balance", newBalance),
	)

	return nil
}

// GetTransactions retrieves wallet transaction history
func (s *WalletService) GetTransactions(ctx context.Context, userID string, limit, offset int) ([]domain.WalletTransaction, error) {
	wallet, err := s.GetWallet(ctx, userID)
	if err != nil {
		return nil, err
	}

	return s.repo.GetTransactions(ctx, wallet.ID, limit, offset)
}

// HasSufficientBalance checks if wallet has enough balance
func (s *WalletService) HasSufficientBalance(ctx context.Context, userID string, amount float64) (bool, error) {
	wallet, err := s.GetWallet(ctx, userID)
	if err != nil {
		return false, err
	}

	return wallet.Balance >= amount, nil
}

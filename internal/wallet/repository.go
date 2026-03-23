package wallet

import (
	"context"
	"errors"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrWalletNotFound    = errors.New("wallet not found")
	ErrInsufficientFunds = errors.New("insufficient funds")
	ErrInvalidOperation  = errors.New("invalid operation type")
)

type Repository interface {
	ApplyOperation(ctx context.Context, walletID uuid.UUID, operationType string, amount int64) (int64, error)
	GetBalance(ctx context.Context, walletID uuid.UUID) (int64, error)
}

type PostgresRepository struct {
	db *pgxpool.Pool
}

func NewPostgresRepository(db *pgxpool.Pool) *PostgresRepository {
	return &PostgresRepository{db: db}
}

func (r *PostgresRepository) ApplyOperation(ctx context.Context, walletID uuid.UUID, operationType string, amount int64) (int64, error) {
	switch strings.ToUpper(operationType) {
	case "DEPOSIT":
		return r.deposit(ctx, walletID, amount)
	case "WITHDRAW":
		return r.withdraw(ctx, walletID, amount)
	default:
		return 0, ErrInvalidOperation
	}
}

func (r *PostgresRepository) GetBalance(ctx context.Context, walletID uuid.UUID) (int64, error) {
	var balance int64

	err := r.db.QueryRow(ctx, `
		SELECT balance
		FROM wallets
		WHERE id = $1
	`, walletID).Scan(&balance)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return 0, ErrWalletNotFound
		}
		return 0, err
	}

	return balance, nil
}

func (r *PostgresRepository) deposit(ctx context.Context, walletID uuid.UUID, amount int64) (int64, error) {
	var newBalance int64

	err := r.db.QueryRow(ctx, `
		INSERT INTO wallets (id, balance, created_at, updated_at)
		VALUES ($1, $2, NOW(), NOW())
		ON CONFLICT (id) DO UPDATE
		SET balance = wallets.balance + EXCLUDED.balance,
		    updated_at = NOW()
		RETURNING balance
	`, walletID, amount).Scan(&newBalance)
	if err != nil {
		return 0, err
	}

	return newBalance, nil
}

func (r *PostgresRepository) withdraw(ctx context.Context, walletID uuid.UUID, amount int64) (int64, error) {
	var newBalance int64

	err := r.db.QueryRow(ctx, `
		UPDATE wallets
		SET balance = balance - $1,
		    updated_at = NOW()
		WHERE id = $2
		  AND balance >= $1
		RETURNING balance
	`, amount, walletID).Scan(&newBalance)
	if err == nil {
		return newBalance, nil
	}

	if !errors.Is(err, pgx.ErrNoRows) {
		return 0, err
	}

	var exists bool
	err = r.db.QueryRow(ctx, `
		SELECT EXISTS(
			SELECT 1
			FROM wallets
			WHERE id = $1
		)
	`, walletID).Scan(&exists)
	if err != nil {
		return 0, err
	}

	if !exists {
		return 0, ErrWalletNotFound
	}

	return 0, ErrInsufficientFunds
}

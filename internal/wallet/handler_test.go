package wallet

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/uuid"
)

type fakeRepo struct {
	applyFunc      func(ctx context.Context, walletID uuid.UUID, operationType string, amount int64) (int64, error)
	getBalanceFunc func(ctx context.Context, walletID uuid.UUID) (int64, error)
}

func (f fakeRepo) ApplyOperation(ctx context.Context, walletID uuid.UUID, operationType string, amount int64) (int64, error) {
	return f.applyFunc(ctx, walletID, operationType, amount)
}

func (f fakeRepo) GetBalance(ctx context.Context, walletID uuid.UUID) (int64, error) {
	return f.getBalanceFunc(ctx, walletID)
}

func TestHandleWalletOperationSuccess(t *testing.T) {
	h := NewHandler(fakeRepo{
		applyFunc: func(ctx context.Context, walletID uuid.UUID, operationType string, amount int64) (int64, error) {
			return 1500, nil
		},
		getBalanceFunc: func(ctx context.Context, walletID uuid.UUID) (int64, error) {
			return 0, nil
		},
	})

	body := `{"walletId":"550e8400-e29b-41d4-a716-446655440000","operationType":"DEPOSIT","amount":1000}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/wallet", strings.NewReader(body))
	w := httptest.NewRecorder()

	h.handleWalletOperation(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}

	var resp walletResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if resp.Balance != 1500 {
		t.Fatalf("expected balance 1500, got %d", resp.Balance)
	}
}

func TestHandleWalletOperationInvalidAmount(t *testing.T) {
	h := NewHandler(fakeRepo{})

	body := `{"walletId":"550e8400-e29b-41d4-a716-446655440000","operationType":"DEPOSIT","amount":0}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/wallet", strings.NewReader(body))
	w := httptest.NewRecorder()

	h.handleWalletOperation(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", w.Code)
	}
}

func TestHandleWalletOperationInsufficientFunds(t *testing.T) {
	h := NewHandler(fakeRepo{
		applyFunc: func(ctx context.Context, walletID uuid.UUID, operationType string, amount int64) (int64, error) {
			return 0, ErrInsufficientFunds
		},
		getBalanceFunc: func(ctx context.Context, walletID uuid.UUID) (int64, error) {
			return 0, nil
		},
	})

	body := `{"walletId":"550e8400-e29b-41d4-a716-446655440000","operationType":"WITHDRAW","amount":1000}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/wallet", strings.NewReader(body))
	w := httptest.NewRecorder()

	h.handleWalletOperation(w, req)

	if w.Code != http.StatusConflict {
		t.Fatalf("expected status 409, got %d", w.Code)
	}
}

func TestHandleGetWalletNotFound(t *testing.T) {
	h := NewHandler(fakeRepo{
		applyFunc: func(ctx context.Context, walletID uuid.UUID, operationType string, amount int64) (int64, error) {
			return 0, nil
		},
		getBalanceFunc: func(ctx context.Context, walletID uuid.UUID) (int64, error) {
			return 0, ErrWalletNotFound
		},
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/wallets/550e8400-e29b-41d4-a716-446655440000", nil)
	w := httptest.NewRecorder()

	h.handleGetWallet(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected status 404, got %d", w.Code)
	}
}

func TestHandleWalletOperationInternalError(t *testing.T) {
	h := NewHandler(fakeRepo{
		applyFunc: func(ctx context.Context, walletID uuid.UUID, operationType string, amount int64) (int64, error) {
			return 0, errors.New("db error")
		},
		getBalanceFunc: func(ctx context.Context, walletID uuid.UUID) (int64, error) {
			return 0, nil
		},
	})

	body := `{"walletId":"550e8400-e29b-41d4-a716-446655440000","operationType":"DEPOSIT","amount":1000}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/wallet", strings.NewReader(body))
	w := httptest.NewRecorder()

	h.handleWalletOperation(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected status 500, got %d", w.Code)
	}
}

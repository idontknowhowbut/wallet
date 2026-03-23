package wallet

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strings"

	"github.com/google/uuid"
)

type Handler struct {
	repo Repository
}

type operationRequest struct {
	WalletID      string `json:"walletId"`
	OperationType string `json:"operationType"`
	Amount        int64  `json:"amount"`
}

type walletResponse struct {
	WalletID string `json:"walletId"`
	Balance  int64  `json:"balance"`
}

type errorResponse struct {
	Error string `json:"error"`
}

func NewHandler(repo Repository) *Handler {
	return &Handler{repo: repo}
}

func (h *Handler) Register(mux *http.ServeMux) {
	mux.HandleFunc("/health", h.handleHealth)
	mux.HandleFunc("/api/v1/wallet", h.handleWalletOperation)
	mux.HandleFunc("/api/v1/wallets/", h.handleGetWallet)
}

func (h *Handler) handleHealth(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (h *Handler) handleWalletOperation(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, errorResponse{Error: "method not allowed"})
		return
	}

	var req operationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "invalid json"})
		return
	}

	if req.Amount <= 0 {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "amount must be greater than 0"})
		return
	}

	walletID, err := uuid.Parse(req.WalletID)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "invalid walletId"})
		return
	}

	balance, err := h.repo.ApplyOperation(r.Context(), walletID, req.OperationType, req.Amount)
	if err != nil {
		h.writeOperationError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, walletResponse{
		WalletID: walletID.String(),
		Balance:  balance,
	})
}

func (h *Handler) handleGetWallet(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, errorResponse{Error: "method not allowed"})
		return
	}

	walletIDStr := strings.TrimPrefix(r.URL.Path, "/api/v1/wallets/")
	if walletIDStr == "" || strings.Contains(walletIDStr, "/") {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "invalid walletID"})
		return
	}

	walletID, err := uuid.Parse(walletIDStr)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "invalid walletID"})
		return
	}

	balance, err := h.repo.GetBalance(r.Context(), walletID)
	if err != nil {
		if errors.Is(err, ErrWalletNotFound) {
			writeJSON(w, http.StatusNotFound, errorResponse{Error: "wallet not found"})
			return
		}

		log.Printf("get wallet error: %v", err)
		writeJSON(w, http.StatusInternalServerError, errorResponse{Error: "internal server error"})
		return
	}

	writeJSON(w, http.StatusOK, walletResponse{
		WalletID: walletID.String(),
		Balance:  balance,
	})
}

func (h *Handler) writeOperationError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, ErrInvalidOperation):
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "operationType must be DEPOSIT or WITHDRAW"})
	case errors.Is(err, ErrWalletNotFound):
		writeJSON(w, http.StatusNotFound, errorResponse{Error: "wallet not found"})
	case errors.Is(err, ErrInsufficientFunds):
		writeJSON(w, http.StatusConflict, errorResponse{Error: "insufficient funds"})
	default:
		log.Printf("wallet operation error: %v", err)
		writeJSON(w, http.StatusInternalServerError, errorResponse{Error: "internal server error"})
	}
}

func writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	if err := json.NewEncoder(w).Encode(data); err != nil {
		log.Printf("json encode error: %v", err)
	}
}

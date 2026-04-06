package handlers

// Extra tests to cover remaining uncovered branches.

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	accountpb "github.com/RAF-SI-2025/EXBanka-4-Backend/shared/pb/account"
	cardpb "github.com/RAF-SI-2025/EXBanka-4-Backend/shared/pb/card"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// ---- DeleteAccount: bad ID ----

func TestDeleteAccount_BadID(t *testing.T) {
	// parseID returns error without writing response; gin defaults to 200 empty body
	w := serveHandler(DeleteAccount(&stubAccountClient{}), "DELETE", "/admin/accounts/:accountId", "/admin/accounts/abc", "")
	if w.Body.Len() != 0 {
		t.Fatalf("expected empty body got %s", w.Body.String())
	}
}

// ---- CreateAccount: limit error (log-only, still 201) ----

func TestCreateAccount_LimitError(t *testing.T) {
	svc := &stubAccountClient{
		createFn: func(_ context.Context, _ *accountpb.CreateAccountRequest, _ ...grpc.CallOption) (*accountpb.CreateAccountResponse, error) {
			return &accountpb.CreateAccountResponse{Account: &accountpb.AccountResponse{Id: 1, AccountNumber: "ACC001"}}, nil
		},
		updateLimitsFn: func(_ context.Context, _ *accountpb.UpdateAccountLimitsRequest, _ ...grpc.CallOption) (*accountpb.UpdateAccountLimitsResponse, error) {
			return nil, fmt.Errorf("limits service down")
		},
	}
	body := `{"clientId":1,"accountType":"CURRENT","currencyCode":"RSD","dailyLimit":1000,"monthlyLimit":5000}`
	w := serveHandlerFull(CreateAccount(svc, &stubCardClient{}), "POST", "/admin/accounts", "/admin/accounts", body, makeClientToken())
	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201 got %d (limit error is non-fatal)", w.Code)
	}
}

// ---- CreateAccount: card creation error (log-only, still 201) ----

func TestCreateAccount_CardCreationError(t *testing.T) {
	svc := &stubAccountClient{
		createFn: func(_ context.Context, _ *accountpb.CreateAccountRequest, _ ...grpc.CallOption) (*accountpb.CreateAccountResponse, error) {
			return &accountpb.CreateAccountResponse{Account: &accountpb.AccountResponse{Id: 1, AccountNumber: "ACC001"}}, nil
		},
	}
	cardSvc := &stubCardClient{
		createCardFn: func(_ context.Context, _ *cardpb.CreateCardRequest, _ ...grpc.CallOption) (*cardpb.CreateCardResponse, error) {
			return nil, fmt.Errorf("card service down")
		},
	}
	body := `{"clientId":1,"accountType":"CURRENT","currencyCode":"RSD","createCard":true}`
	w := serveHandlerFull(CreateAccount(svc, cardSvc), "POST", "/admin/accounts", "/admin/accounts", body, makeClientToken())
	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201 got %d (card error is non-fatal)", w.Code)
	}
}

// ---- CreateAccount: card limit update error (log-only, still 201) ----

func TestCreateAccount_CardLimitError(t *testing.T) {
	svc := &stubAccountClient{
		createFn: func(_ context.Context, _ *accountpb.CreateAccountRequest, _ ...grpc.CallOption) (*accountpb.CreateAccountResponse, error) {
			return &accountpb.CreateAccountResponse{Account: &accountpb.AccountResponse{Id: 1, AccountNumber: "ACC001"}}, nil
		},
	}
	cardSvc := &stubCardClient{
		createCardFn: func(_ context.Context, _ *cardpb.CreateCardRequest, _ ...grpc.CallOption) (*cardpb.CreateCardResponse, error) {
			return &cardpb.CreateCardResponse{Card: &cardpb.CardResponse{CardNumber: "4111111111111111"}}, nil
		},
		updateCardLimitFn: func(_ context.Context, _ *cardpb.UpdateCardLimitRequest, _ ...grpc.CallOption) (*cardpb.UpdateCardLimitResponse, error) {
			return nil, fmt.Errorf("limit service down")
		},
	}
	body := `{"clientId":1,"accountType":"CURRENT","currencyCode":"RSD","createCard":true,"cardLimit":500}`
	w := serveHandlerFull(CreateAccount(svc, cardSvc), "POST", "/admin/accounts", "/admin/accounts", body, makeClientToken())
	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201 got %d (card limit error is non-fatal)", w.Code)
	}
}

// ---- UpdateCardLimit: resolve returns NotFound ----

func TestUpdateCardLimit_ResolveNotFound(t *testing.T) {
	svc := &stubCardClient{
		getCardByIdFn: func(_ context.Context, _ *cardpb.GetCardByIdRequest, _ ...grpc.CallOption) (*cardpb.GetCardByIdResponse, error) {
			return nil, status.Error(codes.NotFound, "card not found")
		},
	}
	w := serveHandler(UpdateCardLimit(svc), "PUT", "/cards/:id/limit", "/cards/1/limit", `{"newLimit":100}`)
	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404 got %d", w.Code)
	}
}

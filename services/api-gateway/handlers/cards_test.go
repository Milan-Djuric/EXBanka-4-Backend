package handlers

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	accountpb "github.com/RAF-SI-2025/EXBanka-4-Backend/shared/pb/account"
	cardpb "github.com/RAF-SI-2025/EXBanka-4-Backend/shared/pb/card"
	clientpb "github.com/RAF-SI-2025/EXBanka-4-Backend/shared/pb/client"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func sampleCardResponse() *cardpb.CardResponse {
	return &cardpb.CardResponse{
		Id:            1,
		CardNumber:    "1234567890123456",
		CardType:      "DEBIT",
		CardName:      "My Card",
		ExpiryDate:    "12/28",
		AccountNumber: "ACC001",
		CardLimit:     1000,
		Status:        "ACTIVE",
		CreatedAt:     "2024-01-01",
	}
}

// ---- cardToJSON ----

func TestCardToJSON(t *testing.T) {
	c := sampleCardResponse()
	h := cardToJSON(c)
	if h["cardNumber"] != "1234567890123456" {
		t.Fatalf("unexpected cardNumber: %v", h["cardNumber"])
	}
}

// ---- GetCardsByAccount ----

func TestGetCardsByAccount_Error(t *testing.T) {
	svc := &stubCardClient{getCardsByAccountFn: func(_ context.Context, _ *cardpb.GetCardsByAccountRequest, _ ...grpc.CallOption) (*cardpb.GetCardsByAccountResponse, error) {
		return nil, fmt.Errorf("rpc error")
	}}
	w := serveHandler(GetCardsByAccount(svc), "GET", "/cards/by-account/:accountNumber", "/cards/by-account/ACC001", "")
	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500 got %d", w.Code)
	}
}

func TestGetCardsByAccount_Happy(t *testing.T) {
	svc := &stubCardClient{getCardsByAccountFn: func(_ context.Context, _ *cardpb.GetCardsByAccountRequest, _ ...grpc.CallOption) (*cardpb.GetCardsByAccountResponse, error) {
		return &cardpb.GetCardsByAccountResponse{Cards: []*cardpb.CardResponse{sampleCardResponse()}}, nil
	}}
	w := serveHandler(GetCardsByAccount(svc), "GET", "/cards/by-account/:accountNumber", "/cards/by-account/ACC001", "")
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 got %d", w.Code)
	}
}

// ---- GetMyCards ----

func TestGetMyCards_NoToken(t *testing.T) {
	w := serveHandler(GetMyCards(&stubAccountClient{}, &stubCardClient{}), "GET", "/cards", "/cards", "")
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 got %d", w.Code)
	}
}

func TestGetMyCards_AccountError(t *testing.T) {
	acctSvc := &stubAccountClient{getMyAccountsFn: func(_ context.Context, _ *accountpb.GetMyAccountsRequest, _ ...grpc.CallOption) (*accountpb.GetMyAccountsResponse, error) {
		return nil, fmt.Errorf("account error")
	}}
	w := serveHandlerFull(GetMyCards(acctSvc, &stubCardClient{}), "GET", "/cards", "/cards", "", makeClientToken())
	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500 got %d", w.Code)
	}
}

func TestGetMyCards_CardError_Skip(t *testing.T) {
	// Card fetch errors should be silently skipped, not fail the whole request
	acctSvc := &stubAccountClient{getMyAccountsFn: func(_ context.Context, _ *accountpb.GetMyAccountsRequest, _ ...grpc.CallOption) (*accountpb.GetMyAccountsResponse, error) {
		return &accountpb.GetMyAccountsResponse{Accounts: []*accountpb.AccountSummary{{AccountNumber: "ACC001"}}}, nil
	}}
	cardSvc := &stubCardClient{getCardsByAccountFn: func(_ context.Context, _ *cardpb.GetCardsByAccountRequest, _ ...grpc.CallOption) (*cardpb.GetCardsByAccountResponse, error) {
		return nil, fmt.Errorf("card error")
	}}
	w := serveHandlerFull(GetMyCards(acctSvc, cardSvc), "GET", "/cards", "/cards", "", makeClientToken())
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 got %d (card errors should be skipped)", w.Code)
	}
}

func TestGetMyCards_Happy(t *testing.T) {
	acctSvc := &stubAccountClient{getMyAccountsFn: func(_ context.Context, _ *accountpb.GetMyAccountsRequest, _ ...grpc.CallOption) (*accountpb.GetMyAccountsResponse, error) {
		return &accountpb.GetMyAccountsResponse{Accounts: []*accountpb.AccountSummary{{AccountNumber: "ACC001"}}}, nil
	}}
	cardSvc := &stubCardClient{getCardsByAccountFn: func(_ context.Context, _ *cardpb.GetCardsByAccountRequest, _ ...grpc.CallOption) (*cardpb.GetCardsByAccountResponse, error) {
		return &cardpb.GetCardsByAccountResponse{Cards: []*cardpb.CardResponse{sampleCardResponse()}}, nil
	}}
	w := serveHandlerFull(GetMyCards(acctSvc, cardSvc), "GET", "/cards", "/cards", "", makeClientToken())
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 got %d", w.Code)
	}
}

// ---- GetCardById ----

func TestGetCardById_NoToken(t *testing.T) {
	w := serveHandler(GetCardById(&stubAccountClient{}, &stubCardClient{}), "GET", "/cards/id/:id", "/cards/id/1", "")
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 got %d", w.Code)
	}
}

func TestGetCardById_BadID(t *testing.T) {
	w := serveHandlerFull(GetCardById(&stubAccountClient{}, &stubCardClient{}), "GET", "/cards/id/:id", "/cards/id/abc", "", makeClientToken())
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 got %d", w.Code)
	}
}

func TestGetCardById_NotFound(t *testing.T) {
	cardSvc := &stubCardClient{getCardByIdFn: func(_ context.Context, _ *cardpb.GetCardByIdRequest, _ ...grpc.CallOption) (*cardpb.GetCardByIdResponse, error) {
		return nil, status.Error(codes.NotFound, "not found")
	}}
	w := serveHandlerFull(GetCardById(&stubAccountClient{}, cardSvc), "GET", "/cards/id/:id", "/cards/id/1", "", makeClientToken())
	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404 got %d", w.Code)
	}
}

func TestGetCardById_CardError(t *testing.T) {
	cardSvc := &stubCardClient{getCardByIdFn: func(_ context.Context, _ *cardpb.GetCardByIdRequest, _ ...grpc.CallOption) (*cardpb.GetCardByIdResponse, error) {
		return nil, fmt.Errorf("internal")
	}}
	w := serveHandlerFull(GetCardById(&stubAccountClient{}, cardSvc), "GET", "/cards/id/:id", "/cards/id/1", "", makeClientToken())
	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500 got %d", w.Code)
	}
}

func TestGetCardById_AccountError(t *testing.T) {
	cardSvc := &stubCardClient{getCardByIdFn: func(_ context.Context, _ *cardpb.GetCardByIdRequest, _ ...grpc.CallOption) (*cardpb.GetCardByIdResponse, error) {
		return &cardpb.GetCardByIdResponse{Card: sampleCardResponse()}, nil
	}}
	acctSvc := &stubAccountClient{getMyAccountsFn: func(_ context.Context, _ *accountpb.GetMyAccountsRequest, _ ...grpc.CallOption) (*accountpb.GetMyAccountsResponse, error) {
		return nil, fmt.Errorf("account error")
	}}
	w := serveHandlerFull(GetCardById(acctSvc, cardSvc), "GET", "/cards/id/:id", "/cards/id/1", "", makeClientToken())
	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500 got %d", w.Code)
	}
}

func TestGetCardById_NotOwned(t *testing.T) {
	cardSvc := &stubCardClient{getCardByIdFn: func(_ context.Context, _ *cardpb.GetCardByIdRequest, _ ...grpc.CallOption) (*cardpb.GetCardByIdResponse, error) {
		c := sampleCardResponse()
		c.AccountNumber = "OTHER_ACC"
		return &cardpb.GetCardByIdResponse{Card: c}, nil
	}}
	acctSvc := &stubAccountClient{getMyAccountsFn: func(_ context.Context, _ *accountpb.GetMyAccountsRequest, _ ...grpc.CallOption) (*accountpb.GetMyAccountsResponse, error) {
		return &accountpb.GetMyAccountsResponse{Accounts: []*accountpb.AccountSummary{{AccountNumber: "ACC001"}}}, nil
	}}
	w := serveHandlerFull(GetCardById(acctSvc, cardSvc), "GET", "/cards/id/:id", "/cards/id/1", "", makeClientToken())
	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403 got %d", w.Code)
	}
}

func TestGetCardById_Happy(t *testing.T) {
	cardSvc := &stubCardClient{getCardByIdFn: func(_ context.Context, _ *cardpb.GetCardByIdRequest, _ ...grpc.CallOption) (*cardpb.GetCardByIdResponse, error) {
		return &cardpb.GetCardByIdResponse{Card: sampleCardResponse()}, nil
	}}
	acctSvc := &stubAccountClient{getMyAccountsFn: func(_ context.Context, _ *accountpb.GetMyAccountsRequest, _ ...grpc.CallOption) (*accountpb.GetMyAccountsResponse, error) {
		return &accountpb.GetMyAccountsResponse{Accounts: []*accountpb.AccountSummary{{AccountNumber: "ACC001"}}}, nil
	}}
	w := serveHandlerFull(GetCardById(acctSvc, cardSvc), "GET", "/cards/id/:id", "/cards/id/1", "", makeClientToken())
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 got %d", w.Code)
	}
}

// ---- GetCardByNumber ----

func TestGetCardByNumber_NoToken(t *testing.T) {
	w := serveHandler(GetCardByNumber(&stubCardClient{}), "GET", "/cards/:number", "/cards/1234", "")
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 got %d", w.Code)
	}
}

func TestGetCardByNumber_NotFound(t *testing.T) {
	svc := &stubCardClient{getCardByNumberFn: func(_ context.Context, _ *cardpb.GetCardByNumberRequest, _ ...grpc.CallOption) (*cardpb.GetCardByNumberResponse, error) {
		return nil, status.Error(codes.NotFound, "not found")
	}}
	w := serveHandlerFull(GetCardByNumber(svc), "GET", "/cards/:number", "/cards/1234", "", makeClientToken())
	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404 got %d", w.Code)
	}
}

func TestGetCardByNumber_Error(t *testing.T) {
	svc := &stubCardClient{getCardByNumberFn: func(_ context.Context, _ *cardpb.GetCardByNumberRequest, _ ...grpc.CallOption) (*cardpb.GetCardByNumberResponse, error) {
		return nil, fmt.Errorf("internal")
	}}
	w := serveHandlerFull(GetCardByNumber(svc), "GET", "/cards/:number", "/cards/1234", "", makeClientToken())
	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500 got %d", w.Code)
	}
}

func TestGetCardByNumber_Happy(t *testing.T) {
	svc := &stubCardClient{getCardByNumberFn: func(_ context.Context, _ *cardpb.GetCardByNumberRequest, _ ...grpc.CallOption) (*cardpb.GetCardByNumberResponse, error) {
		return &cardpb.GetCardByNumberResponse{Card: sampleCardResponse()}, nil
	}}
	w := serveHandlerFull(GetCardByNumber(svc), "GET", "/cards/:number", "/cards/1234", "", makeClientToken())
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 got %d", w.Code)
	}
}

// ---- BlockCard ----

func TestBlockCard_NoToken(t *testing.T) {
	w := serveHandler(BlockCard(&stubCardClient{}), "PUT", "/cards/:id/block", "/cards/1/block", "")
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 got %d", w.Code)
	}
}

func TestBlockCard_BadID(t *testing.T) {
	// non-numeric id causes resolveCardNumber parse error → 400
	w := serveHandlerFull(BlockCard(&stubCardClient{}), "PUT", "/cards/:id/block", "/cards/abc/block", "", makeClientToken())
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 got %d", w.Code)
	}
}

func TestBlockCard_ResolveNotFound(t *testing.T) {
	svc := &stubCardClient{getCardByIdFn: func(_ context.Context, _ *cardpb.GetCardByIdRequest, _ ...grpc.CallOption) (*cardpb.GetCardByIdResponse, error) {
		return nil, status.Error(codes.NotFound, "card not found")
	}}
	w := serveHandlerFull(BlockCard(svc), "PUT", "/cards/:id/block", "/cards/1/block", "", makeClientToken())
	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404 got %d", w.Code)
	}
}

func TestBlockCard_BlockNotFound(t *testing.T) {
	svc := &stubCardClient{
		getCardByIdFn: func(_ context.Context, _ *cardpb.GetCardByIdRequest, _ ...grpc.CallOption) (*cardpb.GetCardByIdResponse, error) {
			return &cardpb.GetCardByIdResponse{Card: sampleCardResponse()}, nil
		},
		blockCardFn: func(_ context.Context, _ *cardpb.BlockCardRequest, _ ...grpc.CallOption) (*cardpb.BlockCardResponse, error) {
			return nil, status.Error(codes.NotFound, "not found")
		},
	}
	w := serveHandlerFull(BlockCard(svc), "PUT", "/cards/:id/block", "/cards/1/block", "", makeClientToken())
	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404 got %d", w.Code)
	}
}

func TestBlockCard_PermissionDenied(t *testing.T) {
	svc := &stubCardClient{
		getCardByIdFn: func(_ context.Context, _ *cardpb.GetCardByIdRequest, _ ...grpc.CallOption) (*cardpb.GetCardByIdResponse, error) {
			return &cardpb.GetCardByIdResponse{Card: sampleCardResponse()}, nil
		},
		blockCardFn: func(_ context.Context, _ *cardpb.BlockCardRequest, _ ...grpc.CallOption) (*cardpb.BlockCardResponse, error) {
			return nil, status.Error(codes.PermissionDenied, "denied")
		},
	}
	w := serveHandlerFull(BlockCard(svc), "PUT", "/cards/:id/block", "/cards/1/block", "", makeClientToken())
	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403 got %d", w.Code)
	}
}

func TestBlockCard_FailedPrecondition(t *testing.T) {
	svc := &stubCardClient{
		getCardByIdFn: func(_ context.Context, _ *cardpb.GetCardByIdRequest, _ ...grpc.CallOption) (*cardpb.GetCardByIdResponse, error) {
			return &cardpb.GetCardByIdResponse{Card: sampleCardResponse()}, nil
		},
		blockCardFn: func(_ context.Context, _ *cardpb.BlockCardRequest, _ ...grpc.CallOption) (*cardpb.BlockCardResponse, error) {
			return nil, status.Error(codes.FailedPrecondition, "already blocked")
		},
	}
	w := serveHandlerFull(BlockCard(svc), "PUT", "/cards/:id/block", "/cards/1/block", "", makeClientToken())
	if w.Code != http.StatusConflict {
		t.Fatalf("expected 409 got %d", w.Code)
	}
}

func TestBlockCard_Error(t *testing.T) {
	svc := &stubCardClient{
		getCardByIdFn: func(_ context.Context, _ *cardpb.GetCardByIdRequest, _ ...grpc.CallOption) (*cardpb.GetCardByIdResponse, error) {
			return &cardpb.GetCardByIdResponse{Card: sampleCardResponse()}, nil
		},
		blockCardFn: func(_ context.Context, _ *cardpb.BlockCardRequest, _ ...grpc.CallOption) (*cardpb.BlockCardResponse, error) {
			return nil, fmt.Errorf("internal")
		},
	}
	w := serveHandlerFull(BlockCard(svc), "PUT", "/cards/:id/block", "/cards/1/block", "", makeClientToken())
	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500 got %d", w.Code)
	}
}

func TestBlockCard_Happy(t *testing.T) {
	svc := &stubCardClient{
		getCardByIdFn: func(_ context.Context, _ *cardpb.GetCardByIdRequest, _ ...grpc.CallOption) (*cardpb.GetCardByIdResponse, error) {
			return &cardpb.GetCardByIdResponse{Card: sampleCardResponse()}, nil
		},
		blockCardFn: func(_ context.Context, _ *cardpb.BlockCardRequest, _ ...grpc.CallOption) (*cardpb.BlockCardResponse, error) {
			return &cardpb.BlockCardResponse{}, nil
		},
	}
	w := serveHandlerFull(BlockCard(svc), "PUT", "/cards/:id/block", "/cards/1/block", "", makeClientToken())
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 got %d", w.Code)
	}
}

// ---- UnblockCard ----

func TestUnblockCard_BadID(t *testing.T) {
	w := serveHandler(UnblockCard(&stubCardClient{}), "PUT", "/cards/:id/unblock", "/cards/abc/unblock", "")
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 got %d", w.Code)
	}
}

func TestUnblockCard_ResolveNotFound(t *testing.T) {
	svc := &stubCardClient{getCardByIdFn: func(_ context.Context, _ *cardpb.GetCardByIdRequest, _ ...grpc.CallOption) (*cardpb.GetCardByIdResponse, error) {
		return nil, status.Error(codes.NotFound, "not found")
	}}
	w := serveHandler(UnblockCard(svc), "PUT", "/cards/:id/unblock", "/cards/1/unblock", "")
	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404 got %d", w.Code)
	}
}

func TestUnblockCard_NotFound(t *testing.T) {
	svc := &stubCardClient{
		getCardByIdFn: func(_ context.Context, _ *cardpb.GetCardByIdRequest, _ ...grpc.CallOption) (*cardpb.GetCardByIdResponse, error) {
			return &cardpb.GetCardByIdResponse{Card: sampleCardResponse()}, nil
		},
		unblockCardFn: func(_ context.Context, _ *cardpb.UnblockCardRequest, _ ...grpc.CallOption) (*cardpb.UnblockCardResponse, error) {
			return nil, status.Error(codes.NotFound, "not found")
		},
	}
	w := serveHandler(UnblockCard(svc), "PUT", "/cards/:id/unblock", "/cards/1/unblock", "")
	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404 got %d", w.Code)
	}
}

func TestUnblockCard_FailedPrecondition(t *testing.T) {
	svc := &stubCardClient{
		getCardByIdFn: func(_ context.Context, _ *cardpb.GetCardByIdRequest, _ ...grpc.CallOption) (*cardpb.GetCardByIdResponse, error) {
			return &cardpb.GetCardByIdResponse{Card: sampleCardResponse()}, nil
		},
		unblockCardFn: func(_ context.Context, _ *cardpb.UnblockCardRequest, _ ...grpc.CallOption) (*cardpb.UnblockCardResponse, error) {
			return nil, status.Error(codes.FailedPrecondition, "not blocked")
		},
	}
	w := serveHandler(UnblockCard(svc), "PUT", "/cards/:id/unblock", "/cards/1/unblock", "")
	if w.Code != http.StatusConflict {
		t.Fatalf("expected 409 got %d", w.Code)
	}
}

func TestUnblockCard_Error(t *testing.T) {
	svc := &stubCardClient{
		getCardByIdFn: func(_ context.Context, _ *cardpb.GetCardByIdRequest, _ ...grpc.CallOption) (*cardpb.GetCardByIdResponse, error) {
			return &cardpb.GetCardByIdResponse{Card: sampleCardResponse()}, nil
		},
		unblockCardFn: func(_ context.Context, _ *cardpb.UnblockCardRequest, _ ...grpc.CallOption) (*cardpb.UnblockCardResponse, error) {
			return nil, fmt.Errorf("internal")
		},
	}
	w := serveHandler(UnblockCard(svc), "PUT", "/cards/:id/unblock", "/cards/1/unblock", "")
	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500 got %d", w.Code)
	}
}

func TestUnblockCard_Happy(t *testing.T) {
	svc := &stubCardClient{
		getCardByIdFn: func(_ context.Context, _ *cardpb.GetCardByIdRequest, _ ...grpc.CallOption) (*cardpb.GetCardByIdResponse, error) {
			return &cardpb.GetCardByIdResponse{Card: sampleCardResponse()}, nil
		},
		unblockCardFn: func(_ context.Context, _ *cardpb.UnblockCardRequest, _ ...grpc.CallOption) (*cardpb.UnblockCardResponse, error) {
			return &cardpb.UnblockCardResponse{}, nil
		},
	}
	w := serveHandler(UnblockCard(svc), "PUT", "/cards/:id/unblock", "/cards/1/unblock", "")
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 got %d", w.Code)
	}
}

// ---- DeactivateCard ----

func TestDeactivateCard_BadID(t *testing.T) {
	w := serveHandler(DeactivateCard(&stubCardClient{}), "PUT", "/cards/:id/deactivate", "/cards/abc/deactivate", "")
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 got %d", w.Code)
	}
}

func TestDeactivateCard_ResolveNotFound(t *testing.T) {
	svc := &stubCardClient{getCardByIdFn: func(_ context.Context, _ *cardpb.GetCardByIdRequest, _ ...grpc.CallOption) (*cardpb.GetCardByIdResponse, error) {
		return nil, status.Error(codes.NotFound, "not found")
	}}
	w := serveHandler(DeactivateCard(svc), "PUT", "/cards/:id/deactivate", "/cards/1/deactivate", "")
	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404 got %d", w.Code)
	}
}

func TestDeactivateCard_NotFound(t *testing.T) {
	svc := &stubCardClient{
		getCardByIdFn: func(_ context.Context, _ *cardpb.GetCardByIdRequest, _ ...grpc.CallOption) (*cardpb.GetCardByIdResponse, error) {
			return &cardpb.GetCardByIdResponse{Card: sampleCardResponse()}, nil
		},
		deactivateCardFn: func(_ context.Context, _ *cardpb.DeactivateCardRequest, _ ...grpc.CallOption) (*cardpb.DeactivateCardResponse, error) {
			return nil, status.Error(codes.NotFound, "not found")
		},
	}
	w := serveHandler(DeactivateCard(svc), "PUT", "/cards/:id/deactivate", "/cards/1/deactivate", "")
	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404 got %d", w.Code)
	}
}

func TestDeactivateCard_FailedPrecondition(t *testing.T) {
	svc := &stubCardClient{
		getCardByIdFn: func(_ context.Context, _ *cardpb.GetCardByIdRequest, _ ...grpc.CallOption) (*cardpb.GetCardByIdResponse, error) {
			return &cardpb.GetCardByIdResponse{Card: sampleCardResponse()}, nil
		},
		deactivateCardFn: func(_ context.Context, _ *cardpb.DeactivateCardRequest, _ ...grpc.CallOption) (*cardpb.DeactivateCardResponse, error) {
			return nil, status.Error(codes.FailedPrecondition, "already deactivated")
		},
	}
	w := serveHandler(DeactivateCard(svc), "PUT", "/cards/:id/deactivate", "/cards/1/deactivate", "")
	if w.Code != http.StatusConflict {
		t.Fatalf("expected 409 got %d", w.Code)
	}
}

func TestDeactivateCard_Error(t *testing.T) {
	svc := &stubCardClient{
		getCardByIdFn: func(_ context.Context, _ *cardpb.GetCardByIdRequest, _ ...grpc.CallOption) (*cardpb.GetCardByIdResponse, error) {
			return &cardpb.GetCardByIdResponse{Card: sampleCardResponse()}, nil
		},
		deactivateCardFn: func(_ context.Context, _ *cardpb.DeactivateCardRequest, _ ...grpc.CallOption) (*cardpb.DeactivateCardResponse, error) {
			return nil, fmt.Errorf("internal")
		},
	}
	w := serveHandler(DeactivateCard(svc), "PUT", "/cards/:id/deactivate", "/cards/1/deactivate", "")
	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500 got %d", w.Code)
	}
}

func TestDeactivateCard_Happy(t *testing.T) {
	svc := &stubCardClient{
		getCardByIdFn: func(_ context.Context, _ *cardpb.GetCardByIdRequest, _ ...grpc.CallOption) (*cardpb.GetCardByIdResponse, error) {
			return &cardpb.GetCardByIdResponse{Card: sampleCardResponse()}, nil
		},
		deactivateCardFn: func(_ context.Context, _ *cardpb.DeactivateCardRequest, _ ...grpc.CallOption) (*cardpb.DeactivateCardResponse, error) {
			return &cardpb.DeactivateCardResponse{}, nil
		},
	}
	w := serveHandler(DeactivateCard(svc), "PUT", "/cards/:id/deactivate", "/cards/1/deactivate", "")
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 got %d", w.Code)
	}
}

// ---- UpdateCardLimit ----

func TestUpdateCardLimit_BadJSON(t *testing.T) {
	w := serveHandler(UpdateCardLimit(&stubCardClient{}), "PUT", "/cards/:id/limit", "/cards/1/limit", `{bad}`)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 got %d", w.Code)
	}
}

func TestUpdateCardLimit_BadID(t *testing.T) {
	w := serveHandler(UpdateCardLimit(&stubCardClient{}), "PUT", "/cards/:id/limit", "/cards/abc/limit", `{"newLimit":100}`)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 got %d", w.Code)
	}
}

func TestUpdateCardLimit_NotFound(t *testing.T) {
	svc := &stubCardClient{
		getCardByIdFn: func(_ context.Context, _ *cardpb.GetCardByIdRequest, _ ...grpc.CallOption) (*cardpb.GetCardByIdResponse, error) {
			return &cardpb.GetCardByIdResponse{Card: sampleCardResponse()}, nil
		},
		updateCardLimitFn: func(_ context.Context, _ *cardpb.UpdateCardLimitRequest, _ ...grpc.CallOption) (*cardpb.UpdateCardLimitResponse, error) {
			return nil, status.Error(codes.NotFound, "not found")
		},
	}
	w := serveHandler(UpdateCardLimit(svc), "PUT", "/cards/:id/limit", "/cards/1/limit", `{"newLimit":100}`)
	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404 got %d", w.Code)
	}
}

func TestUpdateCardLimit_FailedPrecondition(t *testing.T) {
	svc := &stubCardClient{
		getCardByIdFn: func(_ context.Context, _ *cardpb.GetCardByIdRequest, _ ...grpc.CallOption) (*cardpb.GetCardByIdResponse, error) {
			return &cardpb.GetCardByIdResponse{Card: sampleCardResponse()}, nil
		},
		updateCardLimitFn: func(_ context.Context, _ *cardpb.UpdateCardLimitRequest, _ ...grpc.CallOption) (*cardpb.UpdateCardLimitResponse, error) {
			return nil, status.Error(codes.FailedPrecondition, "card inactive")
		},
	}
	w := serveHandler(UpdateCardLimit(svc), "PUT", "/cards/:id/limit", "/cards/1/limit", `{"newLimit":100}`)
	if w.Code != http.StatusConflict {
		t.Fatalf("expected 409 got %d", w.Code)
	}
}

func TestUpdateCardLimit_Error(t *testing.T) {
	svc := &stubCardClient{
		getCardByIdFn: func(_ context.Context, _ *cardpb.GetCardByIdRequest, _ ...grpc.CallOption) (*cardpb.GetCardByIdResponse, error) {
			return &cardpb.GetCardByIdResponse{Card: sampleCardResponse()}, nil
		},
		updateCardLimitFn: func(_ context.Context, _ *cardpb.UpdateCardLimitRequest, _ ...grpc.CallOption) (*cardpb.UpdateCardLimitResponse, error) {
			return nil, fmt.Errorf("internal")
		},
	}
	w := serveHandler(UpdateCardLimit(svc), "PUT", "/cards/:id/limit", "/cards/1/limit", `{"newLimit":100}`)
	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500 got %d", w.Code)
	}
}

func TestUpdateCardLimit_Happy(t *testing.T) {
	svc := &stubCardClient{
		getCardByIdFn: func(_ context.Context, _ *cardpb.GetCardByIdRequest, _ ...grpc.CallOption) (*cardpb.GetCardByIdResponse, error) {
			return &cardpb.GetCardByIdResponse{Card: sampleCardResponse()}, nil
		},
		updateCardLimitFn: func(_ context.Context, _ *cardpb.UpdateCardLimitRequest, _ ...grpc.CallOption) (*cardpb.UpdateCardLimitResponse, error) {
			return &cardpb.UpdateCardLimitResponse{}, nil
		},
	}
	w := serveHandler(UpdateCardLimit(svc), "PUT", "/cards/:id/limit", "/cards/1/limit", `{"newLimit":100}`)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 got %d", w.Code)
	}
}

// ---- InitiateCardRequest ----

func TestInitiateCardRequest_NoToken(t *testing.T) {
	body := `{"accountNumber":"ACC001","cardName":"My Card","forSelf":true}`
	w := serveHandler(InitiateCardRequest(&stubCardClient{}, &stubClientSvcClient{}, &stubEmailClient{}), "POST", "/cards/request", "/cards/request", body)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 got %d", w.Code)
	}
}

func TestInitiateCardRequest_BadJSON(t *testing.T) {
	w := serveHandlerFull(InitiateCardRequest(&stubCardClient{}, &stubClientSvcClient{}, &stubEmailClient{}), "POST", "/cards/request", "/cards/request", `{bad}`, makeClientToken())
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 got %d", w.Code)
	}
}

func TestInitiateCardRequest_ForSelfFalse_NoAuthorizedPerson(t *testing.T) {
	body := `{"accountNumber":"ACC001","cardName":"My Card","forSelf":false}`
	w := serveHandlerFull(InitiateCardRequest(&stubCardClient{}, &stubClientSvcClient{}, &stubEmailClient{}), "POST", "/cards/request", "/cards/request", body, makeClientToken())
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 got %d", w.Code)
	}
}

func TestInitiateCardRequest_InvalidArgument(t *testing.T) {
	svc := &stubCardClient{initiateCardRequestFn: func(_ context.Context, _ *cardpb.InitiateCardRequestRequest, _ ...grpc.CallOption) (*cardpb.InitiateCardRequestResponse, error) {
		return nil, status.Error(codes.InvalidArgument, "invalid account")
	}}
	body := `{"accountNumber":"ACC001","cardName":"My Card","forSelf":true}`
	w := serveHandlerFull(InitiateCardRequest(svc, &stubClientSvcClient{}, &stubEmailClient{}), "POST", "/cards/request", "/cards/request", body, makeClientToken())
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 got %d", w.Code)
	}
}

func TestInitiateCardRequest_ResourceExhausted(t *testing.T) {
	svc := &stubCardClient{initiateCardRequestFn: func(_ context.Context, _ *cardpb.InitiateCardRequestRequest, _ ...grpc.CallOption) (*cardpb.InitiateCardRequestResponse, error) {
		return nil, status.Error(codes.ResourceExhausted, "too many cards")
	}}
	body := `{"accountNumber":"ACC001","cardName":"My Card","forSelf":true}`
	w := serveHandlerFull(InitiateCardRequest(svc, &stubClientSvcClient{}, &stubEmailClient{}), "POST", "/cards/request", "/cards/request", body, makeClientToken())
	if w.Code != http.StatusConflict {
		t.Fatalf("expected 409 got %d", w.Code)
	}
}

func TestInitiateCardRequest_Error(t *testing.T) {
	svc := &stubCardClient{initiateCardRequestFn: func(_ context.Context, _ *cardpb.InitiateCardRequestRequest, _ ...grpc.CallOption) (*cardpb.InitiateCardRequestResponse, error) {
		return nil, fmt.Errorf("internal")
	}}
	body := `{"accountNumber":"ACC001","cardName":"My Card","forSelf":true}`
	w := serveHandlerFull(InitiateCardRequest(svc, &stubClientSvcClient{}, &stubEmailClient{}), "POST", "/cards/request", "/cards/request", body, makeClientToken())
	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500 got %d", w.Code)
	}
}

func TestInitiateCardRequest_ClientFetchError(t *testing.T) {
	// When client fetch fails, email is skipped but request still succeeds
	cardSvc := &stubCardClient{initiateCardRequestFn: func(_ context.Context, _ *cardpb.InitiateCardRequestRequest, _ ...grpc.CallOption) (*cardpb.InitiateCardRequestResponse, error) {
		return &cardpb.InitiateCardRequestResponse{RequestToken: "tok", ConfirmationCode: "123"}, nil
	}}
	clientSvc := &stubClientSvcClient{getByIdFn: func(_ context.Context, _ *clientpb.GetClientByIdRequest, _ ...grpc.CallOption) (*clientpb.GetClientByIdResponse, error) {
		return nil, fmt.Errorf("client not found")
	}}
	body := `{"accountNumber":"ACC001","cardName":"My Card","forSelf":true}`
	w := serveHandlerFull(InitiateCardRequest(cardSvc, clientSvc, &stubEmailClient{}), "POST", "/cards/request", "/cards/request", body, makeClientToken())
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 got %d", w.Code)
	}
}

func TestInitiateCardRequest_Happy_ForOther(t *testing.T) {
	cardSvc := &stubCardClient{initiateCardRequestFn: func(_ context.Context, _ *cardpb.InitiateCardRequestRequest, _ ...grpc.CallOption) (*cardpb.InitiateCardRequestResponse, error) {
		return &cardpb.InitiateCardRequestResponse{RequestToken: "tok", ConfirmationCode: "123"}, nil
	}}
	clientSvc := &stubClientSvcClient{getByIdFn: func(_ context.Context, _ *clientpb.GetClientByIdRequest, _ ...grpc.CallOption) (*clientpb.GetClientByIdResponse, error) {
		return &clientpb.GetClientByIdResponse{Client: &clientpb.Client{Email: "a@b.com", FirstName: "Ana"}}, nil
	}}
	body := `{"accountNumber":"ACC001","cardName":"My Card","forSelf":false,"authorizedPerson":{"firstName":"Bob","lastName":"B","dateOfBirth":"1990-01-01","gender":"M","email":"bob@b.com","phoneNumber":"123","address":"Addr"}}`
	w := serveHandlerFull(InitiateCardRequest(cardSvc, clientSvc, &stubEmailClient{}), "POST", "/cards/request", "/cards/request", body, makeClientToken())
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 got %d", w.Code)
	}
}

// ---- ConfirmCardRequest ----

func TestConfirmCardRequest_NoToken(t *testing.T) {
	w := serveHandler(ConfirmCardRequest(&stubCardClient{}), "POST", "/cards/request/confirm", "/cards/request/confirm", `{"requestToken":"tok","code":"123"}`)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 got %d", w.Code)
	}
}

func TestConfirmCardRequest_BadJSON(t *testing.T) {
	w := serveHandlerFull(ConfirmCardRequest(&stubCardClient{}), "POST", "/cards/request/confirm", "/cards/request/confirm", `{bad}`, makeClientToken())
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 got %d", w.Code)
	}
}

func TestConfirmCardRequest_NotFound(t *testing.T) {
	svc := &stubCardClient{confirmCardRequestFn: func(_ context.Context, _ *cardpb.ConfirmCardRequestRequest, _ ...grpc.CallOption) (*cardpb.ConfirmCardRequestResponse, error) {
		return nil, status.Error(codes.NotFound, "token expired")
	}}
	w := serveHandlerFull(ConfirmCardRequest(svc), "POST", "/cards/request/confirm", "/cards/request/confirm", `{"requestToken":"tok","code":"123"}`, makeClientToken())
	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404 got %d", w.Code)
	}
}

func TestConfirmCardRequest_FailedPrecondition(t *testing.T) {
	svc := &stubCardClient{confirmCardRequestFn: func(_ context.Context, _ *cardpb.ConfirmCardRequestRequest, _ ...grpc.CallOption) (*cardpb.ConfirmCardRequestResponse, error) {
		return nil, status.Error(codes.FailedPrecondition, "already confirmed")
	}}
	w := serveHandlerFull(ConfirmCardRequest(svc), "POST", "/cards/request/confirm", "/cards/request/confirm", `{"requestToken":"tok","code":"123"}`, makeClientToken())
	if w.Code != http.StatusConflict {
		t.Fatalf("expected 409 got %d", w.Code)
	}
}

func TestConfirmCardRequest_PermissionDenied(t *testing.T) {
	svc := &stubCardClient{confirmCardRequestFn: func(_ context.Context, _ *cardpb.ConfirmCardRequestRequest, _ ...grpc.CallOption) (*cardpb.ConfirmCardRequestResponse, error) {
		return nil, status.Error(codes.PermissionDenied, "wrong code")
	}}
	w := serveHandlerFull(ConfirmCardRequest(svc), "POST", "/cards/request/confirm", "/cards/request/confirm", `{"requestToken":"tok","code":"123"}`, makeClientToken())
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 got %d", w.Code)
	}
}

func TestConfirmCardRequest_ResourceExhausted(t *testing.T) {
	svc := &stubCardClient{confirmCardRequestFn: func(_ context.Context, _ *cardpb.ConfirmCardRequestRequest, _ ...grpc.CallOption) (*cardpb.ConfirmCardRequestResponse, error) {
		return nil, status.Error(codes.ResourceExhausted, "too many attempts")
	}}
	w := serveHandlerFull(ConfirmCardRequest(svc), "POST", "/cards/request/confirm", "/cards/request/confirm", `{"requestToken":"tok","code":"123"}`, makeClientToken())
	if w.Code != http.StatusConflict {
		t.Fatalf("expected 409 got %d", w.Code)
	}
}

func TestConfirmCardRequest_Error(t *testing.T) {
	svc := &stubCardClient{confirmCardRequestFn: func(_ context.Context, _ *cardpb.ConfirmCardRequestRequest, _ ...grpc.CallOption) (*cardpb.ConfirmCardRequestResponse, error) {
		return nil, fmt.Errorf("internal")
	}}
	w := serveHandlerFull(ConfirmCardRequest(svc), "POST", "/cards/request/confirm", "/cards/request/confirm", `{"requestToken":"tok","code":"123"}`, makeClientToken())
	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500 got %d", w.Code)
	}
}

func TestConfirmCardRequest_Happy(t *testing.T) {
	svc := &stubCardClient{confirmCardRequestFn: func(_ context.Context, _ *cardpb.ConfirmCardRequestRequest, _ ...grpc.CallOption) (*cardpb.ConfirmCardRequestResponse, error) {
		return &cardpb.ConfirmCardRequestResponse{Card: sampleCardResponse()}, nil
	}}
	w := serveHandlerFull(ConfirmCardRequest(svc), "POST", "/cards/request/confirm", "/cards/request/confirm", `{"requestToken":"tok","code":"123"}`, makeClientToken())
	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201 got %d", w.Code)
	}
}

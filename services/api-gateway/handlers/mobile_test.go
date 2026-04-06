package handlers

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	accountpb "github.com/RAF-SI-2025/EXBanka-4-Backend/shared/pb/account"
	authpb "github.com/RAF-SI-2025/EXBanka-4-Backend/shared/pb/auth"
	paymentpb "github.com/RAF-SI-2025/EXBanka-4-Backend/shared/pb/payment"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// sampleApproval returns a minimal Approval for use in tests.
func sampleApproval() *authpb.Approval {
	return &authpb.Approval{
		Id:         1,
		ActionType: "LOGIN",
		Payload:    "{}",
		Status:     "PENDING",
		ClientId:   1,
		CreatedAt:  "2024-01-01T00:00:00Z",
		ExpiresAt:  "2024-01-01T01:00:00Z",
	}
}

// ---- toApprovalResp ----

func TestToApprovalResp(t *testing.T) {
	a := sampleApproval()
	r := toApprovalResp(a)
	if r.ID != 1 || r.Type != "LOGIN" || r.Status != "PENDING" {
		t.Fatalf("unexpected result: %+v", r)
	}
}

func TestToApprovalResp_InvalidPayload(t *testing.T) {
	a := &authpb.Approval{Id: 2, Payload: "not-json"}
	r := toApprovalResp(a)
	if r.ID != 2 {
		t.Fatalf("unexpected ID: %d", r.ID)
	}
}

// ---- CreateApproval ----

func TestCreateApproval_NoToken(t *testing.T) {
	w := serveHandler(CreateApproval(&stubAuthClient{}), "POST", "/approvals", "/approvals", `{"actionType":"LOGIN","payload":"{}"}`)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 got %d", w.Code)
	}
}

func TestCreateApproval_BadJSON(t *testing.T) {
	w := serveHandlerFull(CreateApproval(&stubAuthClient{}), "POST", "/approvals", "/approvals", `{bad}`, makeClientToken())
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 got %d", w.Code)
	}
}

func TestCreateApproval_Error(t *testing.T) {
	svc := &stubAuthClient{createApprovalFn: func(_ context.Context, _ *authpb.CreateApprovalRequest, _ ...grpc.CallOption) (*authpb.CreateApprovalResponse, error) {
		return nil, fmt.Errorf("internal")
	}}
	w := serveHandlerFull(CreateApproval(svc), "POST", "/approvals", "/approvals", `{"actionType":"LOGIN","payload":"{}"}`, makeClientToken())
	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500 got %d", w.Code)
	}
}

func TestCreateApproval_Happy(t *testing.T) {
	svc := &stubAuthClient{createApprovalFn: func(_ context.Context, _ *authpb.CreateApprovalRequest, _ ...grpc.CallOption) (*authpb.CreateApprovalResponse, error) {
		return &authpb.CreateApprovalResponse{Approval: sampleApproval()}, nil
	}}
	w := serveHandlerFull(CreateApproval(svc), "POST", "/approvals", "/approvals", `{"actionType":"LOGIN","payload":"{}"}`, makeClientToken())
	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201 got %d", w.Code)
	}
}

// ---- PollLoginApproval ----

func TestPollLoginApproval_BadID(t *testing.T) {
	w := serveHandler(PollLoginApproval(&stubAuthClient{}), "GET", "/approvals/:id/poll", "/approvals/abc/poll", "")
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 got %d", w.Code)
	}
}

func TestPollLoginApproval_NotFound(t *testing.T) {
	svc := &stubAuthClient{pollApprovalFn: func(_ context.Context, _ *authpb.PollApprovalRequest, _ ...grpc.CallOption) (*authpb.PollApprovalResponse, error) {
		return nil, status.Error(codes.NotFound, "not found")
	}}
	w := serveHandler(PollLoginApproval(svc), "GET", "/approvals/:id/poll", "/approvals/1/poll", "")
	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404 got %d", w.Code)
	}
}

func TestPollLoginApproval_Error(t *testing.T) {
	svc := &stubAuthClient{pollApprovalFn: func(_ context.Context, _ *authpb.PollApprovalRequest, _ ...grpc.CallOption) (*authpb.PollApprovalResponse, error) {
		return nil, fmt.Errorf("internal")
	}}
	w := serveHandler(PollLoginApproval(svc), "GET", "/approvals/:id/poll", "/approvals/1/poll", "")
	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500 got %d", w.Code)
	}
}

func TestPollLoginApproval_Happy(t *testing.T) {
	svc := &stubAuthClient{pollApprovalFn: func(_ context.Context, _ *authpb.PollApprovalRequest, _ ...grpc.CallOption) (*authpb.PollApprovalResponse, error) {
		return &authpb.PollApprovalResponse{Status: "APPROVED"}, nil
	}}
	w := serveHandler(PollLoginApproval(svc), "GET", "/approvals/:id/poll", "/approvals/1/poll", "")
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 got %d", w.Code)
	}
}

// ---- GetMyApprovals ----

func TestGetMyApprovals_NoToken(t *testing.T) {
	w := serveHandler(GetMyApprovals(&stubAuthClient{}), "GET", "/approvals", "/approvals", "")
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 got %d", w.Code)
	}
}

func TestGetMyApprovals_Error(t *testing.T) {
	svc := &stubAuthClient{getClientApprovalsFn: func(_ context.Context, _ *authpb.GetClientApprovalsRequest, _ ...grpc.CallOption) (*authpb.GetClientApprovalsResponse, error) {
		return nil, fmt.Errorf("internal")
	}}
	w := serveHandlerFull(GetMyApprovals(svc), "GET", "/approvals", "/approvals", "", makeClientToken())
	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500 got %d", w.Code)
	}
}

func TestGetMyApprovals_Happy(t *testing.T) {
	svc := &stubAuthClient{getClientApprovalsFn: func(_ context.Context, _ *authpb.GetClientApprovalsRequest, _ ...grpc.CallOption) (*authpb.GetClientApprovalsResponse, error) {
		return &authpb.GetClientApprovalsResponse{Approvals: []*authpb.Approval{sampleApproval()}}, nil
	}}
	w := serveHandlerFull(GetMyApprovals(svc), "GET", "/approvals", "/approvals", "", makeClientToken())
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 got %d", w.Code)
	}
}

// ---- GetMyApprovalById ----

func TestGetMyApprovalById_NoToken(t *testing.T) {
	w := serveHandler(GetMyApprovalById(&stubAuthClient{}), "GET", "/approvals/:id", "/approvals/1", "")
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 got %d", w.Code)
	}
}

func TestGetMyApprovalById_BadID(t *testing.T) {
	w := serveHandlerFull(GetMyApprovalById(&stubAuthClient{}), "GET", "/approvals/:id", "/approvals/abc", "", makeClientToken())
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 got %d", w.Code)
	}
}

func TestGetMyApprovalById_NotFound(t *testing.T) {
	svc := &stubAuthClient{getApprovalFn: func(_ context.Context, _ *authpb.GetApprovalRequest, _ ...grpc.CallOption) (*authpb.GetApprovalResponse, error) {
		return nil, status.Error(codes.NotFound, "not found")
	}}
	w := serveHandlerFull(GetMyApprovalById(svc), "GET", "/approvals/:id", "/approvals/1", "", makeClientToken())
	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404 got %d", w.Code)
	}
}

func TestGetMyApprovalById_Error(t *testing.T) {
	svc := &stubAuthClient{getApprovalFn: func(_ context.Context, _ *authpb.GetApprovalRequest, _ ...grpc.CallOption) (*authpb.GetApprovalResponse, error) {
		return nil, fmt.Errorf("internal")
	}}
	w := serveHandlerFull(GetMyApprovalById(svc), "GET", "/approvals/:id", "/approvals/1", "", makeClientToken())
	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500 got %d", w.Code)
	}
}

func TestGetMyApprovalById_Forbidden(t *testing.T) {
	// approval belongs to client 99, token has user_id=1
	svc := &stubAuthClient{getApprovalFn: func(_ context.Context, _ *authpb.GetApprovalRequest, _ ...grpc.CallOption) (*authpb.GetApprovalResponse, error) {
		a := sampleApproval()
		a.ClientId = 99
		return &authpb.GetApprovalResponse{Approval: a}, nil
	}}
	w := serveHandlerFull(GetMyApprovalById(svc), "GET", "/approvals/:id", "/approvals/1", "", makeClientToken())
	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403 got %d", w.Code)
	}
}

func TestGetMyApprovalById_Happy(t *testing.T) {
	svc := &stubAuthClient{getApprovalFn: func(_ context.Context, _ *authpb.GetApprovalRequest, _ ...grpc.CallOption) (*authpb.GetApprovalResponse, error) {
		return &authpb.GetApprovalResponse{Approval: sampleApproval()}, nil // ClientId=1, token user_id=1
	}}
	w := serveHandlerFull(GetMyApprovalById(svc), "GET", "/approvals/:id", "/approvals/1", "", makeClientToken())
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 got %d", w.Code)
	}
}

// ---- ApproveApproval ----

func TestApproveApproval_NoToken(t *testing.T) {
	w := serveHandler(ApproveApproval(&stubAuthClient{}, &stubAccountClient{}, &stubPaymentClient{}), "PUT", "/approvals/:id/approve", "/approvals/1/approve", "")
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 got %d", w.Code)
	}
}

func TestApproveApproval_BadID(t *testing.T) {
	w := serveHandlerFull(ApproveApproval(&stubAuthClient{}, &stubAccountClient{}, &stubPaymentClient{}), "PUT", "/approvals/:id/approve", "/approvals/abc/approve", "", makeClientToken())
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 got %d", w.Code)
	}
}

func TestApproveApproval_NotFound(t *testing.T) {
	svc := &stubAuthClient{updateApprovalStatusFn: func(_ context.Context, _ *authpb.UpdateApprovalStatusRequest, _ ...grpc.CallOption) (*authpb.UpdateApprovalStatusResponse, error) {
		return nil, status.Error(codes.NotFound, "not found")
	}}
	w := serveHandlerFull(ApproveApproval(svc, &stubAccountClient{}, &stubPaymentClient{}), "PUT", "/approvals/:id/approve", "/approvals/1/approve", "", makeClientToken())
	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404 got %d", w.Code)
	}
}

func TestApproveApproval_Error(t *testing.T) {
	svc := &stubAuthClient{updateApprovalStatusFn: func(_ context.Context, _ *authpb.UpdateApprovalStatusRequest, _ ...grpc.CallOption) (*authpb.UpdateApprovalStatusResponse, error) {
		return nil, fmt.Errorf("internal")
	}}
	w := serveHandlerFull(ApproveApproval(svc, &stubAccountClient{}, &stubPaymentClient{}), "PUT", "/approvals/:id/approve", "/approvals/1/approve", "", makeClientToken())
	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500 got %d", w.Code)
	}
}

func TestApproveApproval_Happy_Login(t *testing.T) {
	svc := &stubAuthClient{updateApprovalStatusFn: func(_ context.Context, _ *authpb.UpdateApprovalStatusRequest, _ ...grpc.CallOption) (*authpb.UpdateApprovalStatusResponse, error) {
		return &authpb.UpdateApprovalStatusResponse{Approval: sampleApproval()}, nil
	}}
	w := serveHandlerFull(ApproveApproval(svc, &stubAccountClient{}, &stubPaymentClient{}), "PUT", "/approvals/:id/approve", "/approvals/1/approve", "", makeClientToken())
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 got %d", w.Code)
	}
}

func TestApproveApproval_LimitChange(t *testing.T) {
	payload := `{"accountId":5,"dailyLimit":1000,"monthlyLimit":5000}`
	svc := &stubAuthClient{updateApprovalStatusFn: func(_ context.Context, _ *authpb.UpdateApprovalStatusRequest, _ ...grpc.CallOption) (*authpb.UpdateApprovalStatusResponse, error) {
		return &authpb.UpdateApprovalStatusResponse{Approval: &authpb.Approval{
			Id: 1, ActionType: "LIMIT_CHANGE", Payload: payload, Status: "APPROVED", ClientId: 1,
		}}, nil
	}}
	acctSvc := &stubAccountClient{updateLimitsFn: func(_ context.Context, _ *accountpb.UpdateAccountLimitsRequest, _ ...grpc.CallOption) (*accountpb.UpdateAccountLimitsResponse, error) {
		return &accountpb.UpdateAccountLimitsResponse{}, nil
	}}
	w := serveHandlerFull(ApproveApproval(svc, acctSvc, &stubPaymentClient{}), "PUT", "/approvals/:id/approve", "/approvals/1/approve", "", makeClientToken())
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 got %d", w.Code)
	}
}

func TestApproveApproval_Payment(t *testing.T) {
	payload := `{"fromAccount":"acc1","recipientName":"Bob","recipientAccount":"acc2","amount":100}`
	svc := &stubAuthClient{updateApprovalStatusFn: func(_ context.Context, _ *authpb.UpdateApprovalStatusRequest, _ ...grpc.CallOption) (*authpb.UpdateApprovalStatusResponse, error) {
		return &authpb.UpdateApprovalStatusResponse{Approval: &authpb.Approval{
			Id: 1, ActionType: "PAYMENT", Payload: payload, Status: "APPROVED", ClientId: 1,
		}}, nil
	}}
	paymentSvc := &stubPaymentClient{createPaymentFn: func(_ context.Context, _ *paymentpb.CreatePaymentRequest, _ ...grpc.CallOption) (*paymentpb.CreatePaymentResponse, error) {
		return &paymentpb.CreatePaymentResponse{}, nil
	}}
	w := serveHandlerFull(ApproveApproval(svc, &stubAccountClient{}, paymentSvc), "PUT", "/approvals/:id/approve", "/approvals/1/approve", "", makeClientToken())
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 got %d", w.Code)
	}
}

func TestApproveApproval_Transfer(t *testing.T) {
	payload := `{"fromAccount":"acc1","toAccount":"acc2","amount":50}`
	svc := &stubAuthClient{updateApprovalStatusFn: func(_ context.Context, _ *authpb.UpdateApprovalStatusRequest, _ ...grpc.CallOption) (*authpb.UpdateApprovalStatusResponse, error) {
		return &authpb.UpdateApprovalStatusResponse{Approval: &authpb.Approval{
			Id: 1, ActionType: "TRANSFER", Payload: payload, Status: "APPROVED", ClientId: 1,
		}}, nil
	}}
	paymentSvc := &stubPaymentClient{createTransferFn: func(_ context.Context, _ *paymentpb.CreateTransferRequest, _ ...grpc.CallOption) (*paymentpb.CreateTransferResponse, error) {
		return &paymentpb.CreateTransferResponse{}, nil
	}}
	w := serveHandlerFull(ApproveApproval(svc, &stubAccountClient{}, paymentSvc), "PUT", "/approvals/:id/approve", "/approvals/1/approve", "", makeClientToken())
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 got %d", w.Code)
	}
}

// ---- RejectApproval ----

func TestRejectApproval_NoToken(t *testing.T) {
	w := serveHandler(RejectApproval(&stubAuthClient{}), "PUT", "/approvals/:id/reject", "/approvals/1/reject", "")
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 got %d", w.Code)
	}
}

func TestRejectApproval_BadID(t *testing.T) {
	w := serveHandlerFull(RejectApproval(&stubAuthClient{}), "PUT", "/approvals/:id/reject", "/approvals/abc/reject", "", makeClientToken())
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 got %d", w.Code)
	}
}

func TestRejectApproval_NotFound(t *testing.T) {
	svc := &stubAuthClient{updateApprovalStatusFn: func(_ context.Context, _ *authpb.UpdateApprovalStatusRequest, _ ...grpc.CallOption) (*authpb.UpdateApprovalStatusResponse, error) {
		return nil, status.Error(codes.NotFound, "not found")
	}}
	w := serveHandlerFull(RejectApproval(svc), "PUT", "/approvals/:id/reject", "/approvals/1/reject", "", makeClientToken())
	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404 got %d", w.Code)
	}
}

func TestRejectApproval_Error(t *testing.T) {
	svc := &stubAuthClient{updateApprovalStatusFn: func(_ context.Context, _ *authpb.UpdateApprovalStatusRequest, _ ...grpc.CallOption) (*authpb.UpdateApprovalStatusResponse, error) {
		return nil, fmt.Errorf("internal")
	}}
	w := serveHandlerFull(RejectApproval(svc), "PUT", "/approvals/:id/reject", "/approvals/1/reject", "", makeClientToken())
	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500 got %d", w.Code)
	}
}

func TestRejectApproval_Happy(t *testing.T) {
	svc := &stubAuthClient{updateApprovalStatusFn: func(_ context.Context, _ *authpb.UpdateApprovalStatusRequest, _ ...grpc.CallOption) (*authpb.UpdateApprovalStatusResponse, error) {
		return &authpb.UpdateApprovalStatusResponse{Approval: sampleApproval()}, nil
	}}
	w := serveHandlerFull(RejectApproval(svc), "PUT", "/approvals/:id/reject", "/approvals/1/reject", "", makeClientToken())
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 got %d", w.Code)
	}
}

// ---- RegisterMobilePushToken ----

func TestRegisterMobilePushToken_NoToken(t *testing.T) {
	w := serveHandler(RegisterMobilePushToken(&stubAuthClient{}), "POST", "/push-token", "/push-token", `{"token":"abc"}`)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 got %d", w.Code)
	}
}

func TestRegisterMobilePushToken_BadJSON(t *testing.T) {
	w := serveHandlerFull(RegisterMobilePushToken(&stubAuthClient{}), "POST", "/push-token", "/push-token", `{bad}`, makeClientToken())
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 got %d", w.Code)
	}
}

func TestRegisterMobilePushToken_Error(t *testing.T) {
	svc := &stubAuthClient{registerPushTokenFn: func(_ context.Context, _ *authpb.RegisterPushTokenRequest, _ ...grpc.CallOption) (*authpb.RegisterPushTokenResponse, error) {
		return nil, fmt.Errorf("internal")
	}}
	w := serveHandlerFull(RegisterMobilePushToken(svc), "POST", "/push-token", "/push-token", `{"token":"abc"}`, makeClientToken())
	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500 got %d", w.Code)
	}
}

func TestRegisterMobilePushToken_Happy(t *testing.T) {
	svc := &stubAuthClient{registerPushTokenFn: func(_ context.Context, _ *authpb.RegisterPushTokenRequest, _ ...grpc.CallOption) (*authpb.RegisterPushTokenResponse, error) {
		return &authpb.RegisterPushTokenResponse{}, nil
	}}
	w := serveHandlerFull(RegisterMobilePushToken(svc), "POST", "/push-token", "/push-token", `{"token":"abc"}`, makeClientToken())
	if w.Code != http.StatusNoContent {
		t.Fatalf("expected 204 got %d", w.Code)
	}
}

// ---- UnregisterMobilePushToken ----

func TestUnregisterMobilePushToken_NoToken(t *testing.T) {
	w := serveHandler(UnregisterMobilePushToken(&stubAuthClient{}), "DELETE", "/push-token", "/push-token", "")
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 got %d", w.Code)
	}
}

func TestUnregisterMobilePushToken_Error(t *testing.T) {
	svc := &stubAuthClient{unregisterPushTokenFn: func(_ context.Context, _ *authpb.UnregisterPushTokenRequest, _ ...grpc.CallOption) (*authpb.UnregisterPushTokenResponse, error) {
		return nil, fmt.Errorf("internal")
	}}
	w := serveHandlerFull(UnregisterMobilePushToken(svc), "DELETE", "/push-token", "/push-token", "", makeClientToken())
	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500 got %d", w.Code)
	}
}

func TestUnregisterMobilePushToken_Happy(t *testing.T) {
	svc := &stubAuthClient{unregisterPushTokenFn: func(_ context.Context, _ *authpb.UnregisterPushTokenRequest, _ ...grpc.CallOption) (*authpb.UnregisterPushTokenResponse, error) {
		return &authpb.UnregisterPushTokenResponse{}, nil
	}}
	w := serveHandlerFull(UnregisterMobilePushToken(svc), "DELETE", "/push-token", "/push-token", "", makeClientToken())
	if w.Code != http.StatusNoContent {
		t.Fatalf("expected 204 got %d", w.Code)
	}
}

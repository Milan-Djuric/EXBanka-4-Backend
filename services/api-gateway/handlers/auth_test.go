package handlers

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	authpb "github.com/RAF-SI-2025/EXBanka-4-Backend/shared/pb/auth"
	emailpb "github.com/RAF-SI-2025/EXBanka-4-Backend/shared/pb/email"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// stubEmailWithReset extends stubEmailClient to support SendPasswordResetEmail.
type stubEmailWithReset struct {
	stubEmailClient
	sendResetFn func(context.Context, *emailpb.SendPasswordResetEmailRequest, ...grpc.CallOption) (*emailpb.SendPasswordResetEmailResponse, error)
}

func (s *stubEmailWithReset) SendPasswordResetEmail(ctx context.Context, in *emailpb.SendPasswordResetEmailRequest, opts ...grpc.CallOption) (*emailpb.SendPasswordResetEmailResponse, error) {
	if s.sendResetFn != nil {
		return s.sendResetFn(ctx, in, opts...)
	}
	return &emailpb.SendPasswordResetEmailResponse{}, nil
}

// ---- Login ----

func TestLogin_BadJSON(t *testing.T) {
	w := serveHandler(Login(&stubAuthClient{}), "POST", "/login", "/login", `{invalid}`)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestLogin_Error(t *testing.T) {
	client := &stubAuthClient{
		loginFn: func(ctx context.Context, in *authpb.LoginRequest, opts ...grpc.CallOption) (*authpb.LoginResponse, error) {
			return nil, status.Error(codes.Unauthenticated, "bad credentials")
		},
	}
	w := serveHandler(Login(client), "POST", "/login", "/login", `{"email":"a@b.com","password":"wrong"}`)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestLogin_Happy(t *testing.T) {
	client := &stubAuthClient{
		loginFn: func(ctx context.Context, in *authpb.LoginRequest, opts ...grpc.CallOption) (*authpb.LoginResponse, error) {
			return &authpb.LoginResponse{AccessToken: "acc", RefreshToken: "ref"}, nil
		},
	}
	w := serveHandler(Login(client), "POST", "/login", "/login", `{"email":"a@b.com","password":"secret"}`)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "access_token")
}

// ---- Refresh ----

func TestRefresh_BadJSON(t *testing.T) {
	w := serveHandler(Refresh(&stubAuthClient{}), "POST", "/refresh", "/refresh", `{bad}`)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestRefresh_Error(t *testing.T) {
	client := &stubAuthClient{
		refreshFn: func(ctx context.Context, in *authpb.RefreshRequest, opts ...grpc.CallOption) (*authpb.RefreshResponse, error) {
			return nil, fmt.Errorf("expired")
		},
	}
	w := serveHandler(Refresh(client), "POST", "/refresh", "/refresh", `{"refresh_token":"old"}`)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestRefresh_Happy(t *testing.T) {
	client := &stubAuthClient{
		refreshFn: func(ctx context.Context, in *authpb.RefreshRequest, opts ...grpc.CallOption) (*authpb.RefreshResponse, error) {
			return &authpb.RefreshResponse{AccessToken: "newacc"}, nil
		},
	}
	w := serveHandler(Refresh(client), "POST", "/refresh", "/refresh", `{"refresh_token":"valid"}`)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "access_token")
}

// ---- Activate ----

func TestActivate_BadJSON(t *testing.T) {
	w := serveHandler(Activate(&stubAuthClient{}), "POST", "/auth/activate", "/auth/activate", `{bad}`)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestActivate_MissingField(t *testing.T) {
	w := serveHandler(Activate(&stubAuthClient{}), "POST", "/auth/activate", "/auth/activate", `{"token":"t","password":"p"}`)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestActivate_NotFound(t *testing.T) {
	client := &stubAuthClient{
		activateAccountFn: func(ctx context.Context, in *authpb.ActivateAccountRequest, opts ...grpc.CallOption) (*authpb.ActivateAccountResponse, error) {
			return nil, status.Error(codes.NotFound, "token not found")
		},
	}
	w := serveHandler(Activate(client), "POST", "/auth/activate", "/auth/activate", `{"token":"t","password":"p","confirm_password":"p"}`)
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestActivate_FailedPrecondition(t *testing.T) {
	client := &stubAuthClient{
		activateAccountFn: func(ctx context.Context, in *authpb.ActivateAccountRequest, opts ...grpc.CallOption) (*authpb.ActivateAccountResponse, error) {
			return nil, status.Error(codes.FailedPrecondition, "already active")
		},
	}
	w := serveHandler(Activate(client), "POST", "/auth/activate", "/auth/activate", `{"token":"t","password":"p","confirm_password":"p"}`)
	assert.Equal(t, http.StatusConflict, w.Code)
}

func TestActivate_InvalidArgument(t *testing.T) {
	client := &stubAuthClient{
		activateAccountFn: func(ctx context.Context, in *authpb.ActivateAccountRequest, opts ...grpc.CallOption) (*authpb.ActivateAccountResponse, error) {
			return nil, status.Error(codes.InvalidArgument, "passwords don't match")
		},
	}
	w := serveHandler(Activate(client), "POST", "/auth/activate", "/auth/activate", `{"token":"t","password":"p","confirm_password":"p"}`)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestActivate_InternalError(t *testing.T) {
	client := &stubAuthClient{
		activateAccountFn: func(ctx context.Context, in *authpb.ActivateAccountRequest, opts ...grpc.CallOption) (*authpb.ActivateAccountResponse, error) {
			return nil, fmt.Errorf("db error")
		},
	}
	w := serveHandler(Activate(client), "POST", "/auth/activate", "/auth/activate", `{"token":"t","password":"p","confirm_password":"p"}`)
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestActivate_Happy(t *testing.T) {
	client := &stubAuthClient{
		activateAccountFn: func(ctx context.Context, in *authpb.ActivateAccountRequest, opts ...grpc.CallOption) (*authpb.ActivateAccountResponse, error) {
			return &authpb.ActivateAccountResponse{}, nil
		},
	}
	w := serveHandler(Activate(client), "POST", "/auth/activate", "/auth/activate", `{"token":"t","password":"p","confirm_password":"p"}`)
	assert.Equal(t, http.StatusOK, w.Code)
}

// ---- ForgotPassword ----

func TestForgotPassword_MissingEmail(t *testing.T) {
	w := serveHandler(ForgotPassword(&stubAuthClient{}, &stubEmailWithReset{}), "POST", "/auth/forgot-password", "/auth/forgot-password", `{}`)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestForgotPassword_NotFound(t *testing.T) {
	client := &stubAuthClient{
		requestPasswordResetFn: func(ctx context.Context, in *authpb.RequestPasswordResetRequest, opts ...grpc.CallOption) (*authpb.RequestPasswordResetResponse, error) {
			return nil, status.Error(codes.NotFound, "user not found")
		},
	}
	w := serveHandler(ForgotPassword(client, &stubEmailWithReset{}), "POST", "/auth/forgot-password", "/auth/forgot-password", `{"email":"a@b.com"}`)
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestForgotPassword_InternalError(t *testing.T) {
	client := &stubAuthClient{
		requestPasswordResetFn: func(ctx context.Context, in *authpb.RequestPasswordResetRequest, opts ...grpc.CallOption) (*authpb.RequestPasswordResetResponse, error) {
			return nil, fmt.Errorf("db error")
		},
	}
	w := serveHandler(ForgotPassword(client, &stubEmailWithReset{}), "POST", "/auth/forgot-password", "/auth/forgot-password", `{"email":"a@b.com"}`)
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestForgotPassword_Happy(t *testing.T) {
	client := &stubAuthClient{
		requestPasswordResetFn: func(ctx context.Context, in *authpb.RequestPasswordResetRequest, opts ...grpc.CallOption) (*authpb.RequestPasswordResetResponse, error) {
			return &authpb.RequestPasswordResetResponse{Token: "tok", Email: "a@b.com", FirstName: "Ana"}, nil
		},
	}
	w := serveHandler(ForgotPassword(client, &stubEmailWithReset{}), "POST", "/auth/forgot-password", "/auth/forgot-password", `{"email":"a@b.com"}`)
	assert.Equal(t, http.StatusOK, w.Code)
}

// ---- ClientLogin ----

func TestClientLogin_BadJSON(t *testing.T) {
	w := serveHandler(ClientLogin(&stubAuthClient{}), "POST", "/client/login", "/client/login", `{bad}`)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestClientLogin_Error(t *testing.T) {
	client := &stubAuthClient{
		clientLoginFn: func(ctx context.Context, in *authpb.ClientLoginRequest, opts ...grpc.CallOption) (*authpb.ClientLoginResponse, error) {
			return nil, status.Error(codes.Unauthenticated, "bad credentials")
		},
	}
	w := serveHandler(ClientLogin(client), "POST", "/client/login", "/client/login", `{"email":"a@b.com","password":"wrong"}`)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestClientLogin_HappyWithTokens(t *testing.T) {
	client := &stubAuthClient{
		clientLoginFn: func(ctx context.Context, in *authpb.ClientLoginRequest, opts ...grpc.CallOption) (*authpb.ClientLoginResponse, error) {
			return &authpb.ClientLoginResponse{AccessToken: "acc", RefreshToken: "ref", ApprovalRequestId: 0}, nil
		},
	}
	w := serveHandler(ClientLogin(client), "POST", "/client/login", "/client/login", `{"email":"a@b.com","password":"pass"}`)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "access_token")
}

func TestClientLogin_HappyWithApproval(t *testing.T) {
	client := &stubAuthClient{
		clientLoginFn: func(ctx context.Context, in *authpb.ClientLoginRequest, opts ...grpc.CallOption) (*authpb.ClientLoginResponse, error) {
			return &authpb.ClientLoginResponse{ApprovalRequestId: 42}, nil
		},
	}
	w := serveHandler(ClientLogin(client), "POST", "/client/login", "/client/login", `{"email":"a@b.com","password":"pass","source":"mobile"}`)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "approvalRequestId")
}

// ---- ClientRefresh ----

func TestClientRefresh_BadJSON(t *testing.T) {
	w := serveHandler(ClientRefresh(&stubAuthClient{}), "POST", "/client/refresh", "/client/refresh", `{bad}`)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestClientRefresh_Error(t *testing.T) {
	client := &stubAuthClient{
		clientRefreshFn: func(ctx context.Context, in *authpb.ClientRefreshRequest, opts ...grpc.CallOption) (*authpb.ClientRefreshResponse, error) {
			return nil, fmt.Errorf("expired")
		},
	}
	w := serveHandler(ClientRefresh(client), "POST", "/client/refresh", "/client/refresh", `{"refresh_token":"old"}`)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestClientRefresh_Happy(t *testing.T) {
	client := &stubAuthClient{
		clientRefreshFn: func(ctx context.Context, in *authpb.ClientRefreshRequest, opts ...grpc.CallOption) (*authpb.ClientRefreshResponse, error) {
			return &authpb.ClientRefreshResponse{AccessToken: "newacc"}, nil
		},
	}
	w := serveHandler(ClientRefresh(client), "POST", "/client/refresh", "/client/refresh", `{"refresh_token":"valid"}`)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "access_token")
}

// ---- ActivateClient ----

func TestActivateClient_BadJSON(t *testing.T) {
	w := serveHandler(ActivateClient(&stubAuthClient{}), "POST", "/client/activate", "/client/activate", `{bad}`)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestActivateClient_NotFound(t *testing.T) {
	client := &stubAuthClient{
		activateClientFn: func(ctx context.Context, in *authpb.ActivateClientRequest, opts ...grpc.CallOption) (*authpb.ActivateClientResponse, error) {
			return nil, status.Error(codes.NotFound, "token not found")
		},
	}
	w := serveHandler(ActivateClient(client), "POST", "/client/activate", "/client/activate", `{"token":"t","password":"p","confirm_password":"p"}`)
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestActivateClient_FailedPrecondition(t *testing.T) {
	client := &stubAuthClient{
		activateClientFn: func(ctx context.Context, in *authpb.ActivateClientRequest, opts ...grpc.CallOption) (*authpb.ActivateClientResponse, error) {
			return nil, status.Error(codes.FailedPrecondition, "already active")
		},
	}
	w := serveHandler(ActivateClient(client), "POST", "/client/activate", "/client/activate", `{"token":"t","password":"p","confirm_password":"p"}`)
	assert.Equal(t, http.StatusConflict, w.Code)
}

func TestActivateClient_InvalidArgument(t *testing.T) {
	client := &stubAuthClient{
		activateClientFn: func(ctx context.Context, in *authpb.ActivateClientRequest, opts ...grpc.CallOption) (*authpb.ActivateClientResponse, error) {
			return nil, status.Error(codes.InvalidArgument, "mismatch")
		},
	}
	w := serveHandler(ActivateClient(client), "POST", "/client/activate", "/client/activate", `{"token":"t","password":"p","confirm_password":"p"}`)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestActivateClient_InternalError(t *testing.T) {
	client := &stubAuthClient{
		activateClientFn: func(ctx context.Context, in *authpb.ActivateClientRequest, opts ...grpc.CallOption) (*authpb.ActivateClientResponse, error) {
			return nil, fmt.Errorf("db error")
		},
	}
	w := serveHandler(ActivateClient(client), "POST", "/client/activate", "/client/activate", `{"token":"t","password":"p","confirm_password":"p"}`)
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestActivateClient_Happy(t *testing.T) {
	client := &stubAuthClient{
		activateClientFn: func(ctx context.Context, in *authpb.ActivateClientRequest, opts ...grpc.CallOption) (*authpb.ActivateClientResponse, error) {
			return &authpb.ActivateClientResponse{}, nil
		},
	}
	w := serveHandler(ActivateClient(client), "POST", "/client/activate", "/client/activate", `{"token":"t","password":"p","confirm_password":"p"}`)
	assert.Equal(t, http.StatusOK, w.Code)
}

// ---- ResetPassword ----

func TestResetPassword_BadJSON(t *testing.T) {
	w := serveHandler(ResetPassword(&stubAuthClient{}), "POST", "/auth/reset-password", "/auth/reset-password", `{bad}`)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestResetPassword_NotFound(t *testing.T) {
	client := &stubAuthClient{
		resetPasswordFn: func(ctx context.Context, in *authpb.ResetPasswordRequest, opts ...grpc.CallOption) (*authpb.ResetPasswordResponse, error) {
			return nil, status.Error(codes.NotFound, "token not found")
		},
	}
	w := serveHandler(ResetPassword(client), "POST", "/auth/reset-password", "/auth/reset-password", `{"token":"t","password":"p","confirm_password":"p"}`)
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestResetPassword_FailedPrecondition(t *testing.T) {
	client := &stubAuthClient{
		resetPasswordFn: func(ctx context.Context, in *authpb.ResetPasswordRequest, opts ...grpc.CallOption) (*authpb.ResetPasswordResponse, error) {
			return nil, status.Error(codes.FailedPrecondition, "already used")
		},
	}
	w := serveHandler(ResetPassword(client), "POST", "/auth/reset-password", "/auth/reset-password", `{"token":"t","password":"p","confirm_password":"p"}`)
	assert.Equal(t, http.StatusConflict, w.Code)
}

func TestResetPassword_InvalidArgument(t *testing.T) {
	client := &stubAuthClient{
		resetPasswordFn: func(ctx context.Context, in *authpb.ResetPasswordRequest, opts ...grpc.CallOption) (*authpb.ResetPasswordResponse, error) {
			return nil, status.Error(codes.InvalidArgument, "mismatch")
		},
	}
	w := serveHandler(ResetPassword(client), "POST", "/auth/reset-password", "/auth/reset-password", `{"token":"t","password":"p","confirm_password":"p"}`)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestResetPassword_InternalError(t *testing.T) {
	client := &stubAuthClient{
		resetPasswordFn: func(ctx context.Context, in *authpb.ResetPasswordRequest, opts ...grpc.CallOption) (*authpb.ResetPasswordResponse, error) {
			return nil, fmt.Errorf("db error")
		},
	}
	w := serveHandler(ResetPassword(client), "POST", "/auth/reset-password", "/auth/reset-password", `{"token":"t","password":"p","confirm_password":"p"}`)
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestResetPassword_Happy(t *testing.T) {
	client := &stubAuthClient{
		resetPasswordFn: func(ctx context.Context, in *authpb.ResetPasswordRequest, opts ...grpc.CallOption) (*authpb.ResetPasswordResponse, error) {
			return &authpb.ResetPasswordResponse{}, nil
		},
	}
	w := serveHandler(ResetPassword(client), "POST", "/auth/reset-password", "/auth/reset-password", `{"token":"t","password":"p","confirm_password":"p"}`)
	assert.Equal(t, http.StatusOK, w.Code)
}

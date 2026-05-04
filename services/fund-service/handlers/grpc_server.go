package handlers

import (
	"context"
	"database/sql"
	"strings"
	"time"

	pb_account "github.com/RAF-SI-2025/EXBanka-4-Backend/shared/pb/account"
	pb "github.com/RAF-SI-2025/EXBanka-4-Backend/shared/pb/fund"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type FundServer struct {
	pb.UnimplementedFundServiceServer
	DB            *sql.DB // fund_db
	AccountDB     *sql.DB // account_db
	EmployeeDB    *sql.DB // employee_db
	AccountClient pb_account.AccountServiceClient
}

func (s *FundServer) Ping(_ context.Context, _ *pb.PingRequest) (*pb.PingResponse, error) {
	return &pb.PingResponse{Message: "fund-service ok"}, nil
}

func (s *FundServer) CreateFund(ctx context.Context, req *pb.CreateFundRequest) (*pb.FundResponse, error) {
	if req.Name == "" {
		return nil, status.Error(codes.InvalidArgument, "name is required")
	}
	if req.ManagerId == 0 {
		return nil, status.Error(codes.InvalidArgument, "manager_id is required")
	}

	// Create a bank account for this fund
	accountResp, err := s.AccountClient.CreateAccount(ctx, &pb_account.CreateAccountRequest{
		ClientId:       0,
		AccountType:    "BANK",
		CurrencyCode:   "RSD",
		InitialBalance: 0,
		AccountName:    "Fund: " + req.Name,
		CreateCard:     false,
		EmployeeId:     req.CreatedById,
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to create fund account: %v", err)
	}

	accountID := accountResp.GetAccount().GetId()

	var id int64
	err = s.DB.QueryRowContext(ctx, `
		INSERT INTO investment_funds (name, description, minimum_contribution, manager_id, account_id)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id`,
		req.Name, req.Description, req.MinimumContribution, req.ManagerId, accountID,
	).Scan(&id)
	if err != nil {
		if isUniqueViolation(err) {
			return nil, status.Errorf(codes.AlreadyExists, "fund with name %q already exists", req.Name)
		}
		return nil, status.Errorf(codes.Internal, "failed to create fund: %v", err)
	}

	return s.fetchFundByID(ctx, id, true)
}

func (s *FundServer) ListFunds(ctx context.Context, req *pb.ListFundsRequest) (*pb.ListFundsResponse, error) {
	query := `SELECT id FROM investment_funds WHERE active = true`
	args := []interface{}{}

	if req.ManagerIdFilter != 0 {
		args = append(args, req.ManagerIdFilter)
		query += ` AND manager_id = $1`
	}
	query += ` ORDER BY name ASC`

	rows, err := s.DB.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to list funds: %v", err)
	}
	defer func() { _ = rows.Close() }()

	var ids []int64
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, status.Errorf(codes.Internal, "failed to scan fund id: %v", err)
		}
		ids = append(ids, id)
	}

	funds := make([]*pb.FundResponse, 0, len(ids))
	for _, id := range ids {
		f, err := s.fetchFundByID(ctx, id, false)
		if err != nil {
			return nil, err
		}
		funds = append(funds, f)
	}
	return &pb.ListFundsResponse{Funds: funds}, nil
}

func (s *FundServer) GetFund(ctx context.Context, req *pb.GetFundRequest) (*pb.FundResponse, error) {
	return s.fetchFundByID(ctx, req.Id, true)
}

func (s *FundServer) UpdateFund(ctx context.Context, req *pb.UpdateFundRequest) (*pb.FundResponse, error) {
	if req.Id == 0 {
		return nil, status.Error(codes.InvalidArgument, "id is required")
	}

	_, err := s.DB.ExecContext(ctx, `
		UPDATE investment_funds
		SET name = $1, description = $2, minimum_contribution = $3, manager_id = $4
		WHERE id = $5 AND active = true`,
		req.Name, req.Description, req.MinimumContribution, req.ManagerId, req.Id,
	)
	if err != nil {
		if isUniqueViolation(err) {
			return nil, status.Errorf(codes.AlreadyExists, "fund with name %q already exists", req.Name)
		}
		return nil, status.Errorf(codes.Internal, "failed to update fund: %v", err)
	}

	return s.fetchFundByID(ctx, req.Id, true)
}

func (s *FundServer) DeleteFund(ctx context.Context, req *pb.DeleteFundRequest) (*pb.DeleteFundResponse, error) {
	if req.Id == 0 {
		return nil, status.Error(codes.InvalidArgument, "id is required")
	}

	// Check for active positions
	var count int64
	err := s.DB.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM client_fund_positions
		WHERE fund_id = $1 AND total_invested_amount > 0`, req.Id,
	).Scan(&count)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to check fund positions: %v", err)
	}
	if count > 0 {
		return nil, status.Error(codes.PermissionDenied, "cannot delete fund with active client positions")
	}

	res, err := s.DB.ExecContext(ctx, `UPDATE investment_funds SET active = false WHERE id = $1 AND active = true`, req.Id)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to delete fund: %v", err)
	}
	rows, _ := res.RowsAffected()
	if rows == 0 {
		return nil, status.Error(codes.NotFound, "fund not found")
	}

	return &pb.DeleteFundResponse{}, nil
}

func (s *FundServer) fetchFundByID(ctx context.Context, id int64, includeAccountNumber bool) (*pb.FundResponse, error) {
	var f pb.FundResponse
	var description sql.NullString
	var accountID sql.NullInt64
	var createdAt time.Time

	err := s.DB.QueryRowContext(ctx, `
		SELECT id, name, description, minimum_contribution, manager_id,
		       liquid_assets, account_id, created_at, active
		FROM investment_funds WHERE id = $1`, id,
	).Scan(
		&f.Id, &f.Name, &description, &f.MinimumContribution, &f.ManagerId,
		&f.LiquidAssets, &accountID, &createdAt, &f.Active,
	)
	if err == sql.ErrNoRows {
		return nil, status.Error(codes.NotFound, "fund not found")
	} else if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to fetch fund: %v", err)
	}

	if description.Valid {
		f.Description = description.String
	}
	if accountID.Valid {
		f.AccountId = accountID.Int64
	}
	f.CreatedAt = createdAt.Format(time.RFC3339)

	// fund_value = liquid_assets (no portfolio positions in Sprint 1)
	f.FundValue = f.LiquidAssets

	// profit = fund_value - total invested
	var totalInvested float64
	_ = s.DB.QueryRowContext(ctx, `
		SELECT COALESCE(SUM(total_invested_amount), 0)
		FROM client_fund_positions WHERE fund_id = $1`, id,
	).Scan(&totalInvested)
	f.Profit = f.FundValue - totalInvested

	// manager name
	if f.ManagerId != 0 {
		var managerName string
		_ = s.EmployeeDB.QueryRowContext(ctx,
			`SELECT first_name || ' ' || last_name FROM employees WHERE id = $1`, f.ManagerId,
		).Scan(&managerName)
		f.ManagerName = managerName
	}

	// account number (only needed for GetFund, not list)
	if includeAccountNumber && f.AccountId != 0 {
		var accountNumber string
		_ = s.AccountDB.QueryRowContext(ctx,
			`SELECT account_number FROM accounts WHERE id = $1`, f.AccountId,
		).Scan(&accountNumber)
		f.AccountNumber = accountNumber
	}

	return &f, nil
}

// isUniqueViolation checks if the error is a PostgreSQL unique constraint violation (error code 23505).
func isUniqueViolation(err error) bool {
	return strings.Contains(err.Error(), "23505") || strings.Contains(err.Error(), "unique constraint")
}

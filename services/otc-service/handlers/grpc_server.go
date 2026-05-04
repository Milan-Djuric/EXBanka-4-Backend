package handlers

import (
	"context"
	"database/sql"
	"time"

	pb "github.com/RAF-SI-2025/EXBanka-4-Backend/shared/pb/otc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type OtcServer struct {
	pb.UnimplementedOtcServiceServer
	DB         *sql.DB // otc_db
	EmployeeDB *sql.DB // employee_db
	ClientDB   *sql.DB // client_db
}

// getUserName looks up the display name for a user in the appropriate DB.
func getUserName(employeeDB, clientDB *sql.DB, userID int64, userType string) string {
	if userID == 0 {
		return ""
	}
	var name string
	var err error
	if userType == "EMPLOYEE" {
		err = employeeDB.QueryRow(`SELECT first_name || ' ' || last_name FROM employees WHERE id = $1`, userID).Scan(&name)
	} else {
		err = clientDB.QueryRow(`SELECT first_name || ' ' || last_name FROM clients WHERE id = $1`, userID).Scan(&name)
	}
	if err != nil {
		return ""
	}
	return name
}

func (s *OtcServer) Ping(_ context.Context, _ *pb.PingRequest) (*pb.PingResponse, error) {
	return &pb.PingResponse{Message: "otc-service ok"}, nil
}

func (s *OtcServer) CreateNegotiation(ctx context.Context, req *pb.CreateNegotiationRequest) (*pb.NegotiationResponse, error) {
	if req.Ticker == "" {
		return nil, status.Error(codes.InvalidArgument, "ticker is required")
	}
	if req.Amount <= 0 {
		return nil, status.Error(codes.InvalidArgument, "amount must be positive")
	}
	if req.PricePerStock <= 0 {
		return nil, status.Error(codes.InvalidArgument, "price_per_stock must be positive")
	}

	now := time.Now()

	var id int64
	err := s.DB.QueryRowContext(ctx, `
		INSERT INTO otc_negotiations
			(ticker, seller_id, seller_type, buyer_id, buyer_type,
			 amount, price_per_stock, settlement_date, premium, currency,
			 last_modified, modified_by_id, modified_by_type, status)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, 'PENDING_SELLER')
		RETURNING id`,
		req.Ticker, req.SellerId, req.SellerType, req.BuyerId, req.BuyerType,
		req.Amount, req.PricePerStock, req.SettlementDate, req.Premium, req.Currency,
		now, req.BuyerId, req.BuyerType,
	).Scan(&id)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to create negotiation: %v", err)
	}

	return s.fetchNegotiationByID(ctx, id)
}

func (s *OtcServer) ListNegotiations(ctx context.Context, req *pb.ListNegotiationsRequest) (*pb.ListNegotiationsResponse, error) {
	rows, err := s.DB.QueryContext(ctx, `
		SELECT id FROM otc_negotiations
		WHERE (seller_id = $1 AND seller_type = $2)
		   OR (buyer_id  = $1 AND buyer_type  = $2)
		ORDER BY last_modified DESC`,
		req.CallerId, req.CallerType,
	)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to list negotiations: %v", err)
	}
	defer func() { _ = rows.Close() }()

	var ids []int64
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, status.Errorf(codes.Internal, "failed to scan id: %v", err)
		}
		ids = append(ids, id)
	}

	negotiations := make([]*pb.NegotiationResponse, 0, len(ids))
	for _, id := range ids {
		neg, err := s.fetchNegotiationByID(ctx, id)
		if err != nil {
			return nil, err
		}
		negotiations = append(negotiations, neg)
	}
	return &pb.ListNegotiationsResponse{Negotiations: negotiations}, nil
}

func (s *OtcServer) GetNegotiation(ctx context.Context, req *pb.GetNegotiationRequest) (*pb.NegotiationResponse, error) {
	return s.fetchNegotiationByID(ctx, req.NegotiationId)
}

func (s *OtcServer) CounterOffer(ctx context.Context, req *pb.CounterOfferRequest) (*pb.NegotiationResponse, error) {
	// Load current state
	var sellerID, buyerID int64
	var sellerType, buyerType, currentStatus string
	err := s.DB.QueryRowContext(ctx, `
		SELECT seller_id, seller_type, buyer_id, buyer_type, status
		FROM otc_negotiations WHERE id = $1`, req.NegotiationId,
	).Scan(&sellerID, &sellerType, &buyerID, &buyerType, &currentStatus)
	if err == sql.ErrNoRows {
		return nil, status.Error(codes.NotFound, "negotiation not found")
	} else if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to load negotiation: %v", err)
	}

	// Check caller is participant
	isSeller := req.CallerId == sellerID && req.CallerType == sellerType
	isBuyer := req.CallerId == buyerID && req.CallerType == buyerType
	if !isSeller && !isBuyer {
		return nil, status.Error(codes.PermissionDenied, "caller is not a participant in this negotiation")
	}

	// Check it's caller's turn
	if currentStatus == "PENDING_SELLER" && !isSeller {
		return nil, status.Error(codes.AlreadyExists, "not your turn: waiting for seller")
	}
	if currentStatus == "PENDING_BUYER" && !isBuyer {
		return nil, status.Error(codes.AlreadyExists, "not your turn: waiting for buyer")
	}
	if currentStatus != "PENDING_SELLER" && currentStatus != "PENDING_BUYER" {
		return nil, status.Errorf(codes.FailedPrecondition, "negotiation is in terminal state: %s", currentStatus)
	}

	// Flip status
	newStatus := "PENDING_BUYER"
	if isBuyer {
		newStatus = "PENDING_SELLER"
	}

	now := time.Now()
	_, err = s.DB.ExecContext(ctx, `
		UPDATE otc_negotiations
		SET amount = $1, price_per_stock = $2, settlement_date = $3, premium = $4,
		    last_modified = $5, modified_by_id = $6, modified_by_type = $7, status = $8
		WHERE id = $9`,
		req.Amount, req.PricePerStock, req.SettlementDate, req.Premium,
		now, req.CallerId, req.CallerType, newStatus,
		req.NegotiationId,
	)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to update negotiation: %v", err)
	}

	return s.fetchNegotiationByID(ctx, req.NegotiationId)
}

func (s *OtcServer) AcceptNegotiation(ctx context.Context, req *pb.AcceptNegotiationRequest) (*pb.NegotiationResponse, error) {
	var sellerID, buyerID int64
	var sellerType, buyerType, currentStatus string
	err := s.DB.QueryRowContext(ctx, `
		SELECT seller_id, seller_type, buyer_id, buyer_type, status
		FROM otc_negotiations WHERE id = $1`, req.NegotiationId,
	).Scan(&sellerID, &sellerType, &buyerID, &buyerType, &currentStatus)
	if err == sql.ErrNoRows {
		return nil, status.Error(codes.NotFound, "negotiation not found")
	} else if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to load negotiation: %v", err)
	}

	isSeller := req.CallerId == sellerID && req.CallerType == sellerType
	isBuyer := req.CallerId == buyerID && req.CallerType == buyerType
	if !isSeller && !isBuyer {
		return nil, status.Error(codes.PermissionDenied, "caller is not a participant in this negotiation")
	}

	// Check it's caller's turn
	if currentStatus == "PENDING_SELLER" && !isSeller {
		return nil, status.Error(codes.AlreadyExists, "not your turn: waiting for seller")
	}
	if currentStatus == "PENDING_BUYER" && !isBuyer {
		return nil, status.Error(codes.AlreadyExists, "not your turn: waiting for buyer")
	}
	if currentStatus != "PENDING_SELLER" && currentStatus != "PENDING_BUYER" {
		return nil, status.Errorf(codes.FailedPrecondition, "negotiation is in terminal state: %s", currentStatus)
	}

	now := time.Now()
	_, err = s.DB.ExecContext(ctx, `
		UPDATE otc_negotiations
		SET status = 'ACCEPTED', last_modified = $1, modified_by_id = $2, modified_by_type = $3
		WHERE id = $4`,
		now, req.CallerId, req.CallerType, req.NegotiationId,
	)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to accept negotiation: %v", err)
	}

	return s.fetchNegotiationByID(ctx, req.NegotiationId)
}

func (s *OtcServer) RejectNegotiation(ctx context.Context, req *pb.RejectNegotiationRequest) (*pb.NegotiationResponse, error) {
	var sellerID, buyerID int64
	var sellerType, buyerType, currentStatus string
	err := s.DB.QueryRowContext(ctx, `
		SELECT seller_id, seller_type, buyer_id, buyer_type, status
		FROM otc_negotiations WHERE id = $1`, req.NegotiationId,
	).Scan(&sellerID, &sellerType, &buyerID, &buyerType, &currentStatus)
	if err == sql.ErrNoRows {
		return nil, status.Error(codes.NotFound, "negotiation not found")
	} else if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to load negotiation: %v", err)
	}

	isSeller := req.CallerId == sellerID && req.CallerType == sellerType
	isBuyer := req.CallerId == buyerID && req.CallerType == buyerType
	if !isSeller && !isBuyer {
		return nil, status.Error(codes.PermissionDenied, "caller is not a participant in this negotiation")
	}

	if currentStatus == "ACCEPTED" || currentStatus == "REJECTED" {
		return nil, status.Errorf(codes.FailedPrecondition, "negotiation is already in terminal state: %s", currentStatus)
	}

	now := time.Now()
	_, err = s.DB.ExecContext(ctx, `
		UPDATE otc_negotiations
		SET status = 'REJECTED', last_modified = $1, modified_by_id = $2, modified_by_type = $3
		WHERE id = $4`,
		now, req.CallerId, req.CallerType, req.NegotiationId,
	)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to reject negotiation: %v", err)
	}

	return s.fetchNegotiationByID(ctx, req.NegotiationId)
}

// fetchNegotiationByID loads a single negotiation by ID and enriches it with user names.
func (s *OtcServer) fetchNegotiationByID(ctx context.Context, id int64) (*pb.NegotiationResponse, error) {
	var n pb.NegotiationResponse
	var lastModified time.Time
	var settlementDate string
	var modifiedByID sql.NullInt64
	var modifiedByType sql.NullString

	err := s.DB.QueryRowContext(ctx, `
		SELECT id, ticker, seller_id, seller_type, buyer_id, buyer_type,
		       amount, price_per_stock, settlement_date::text, premium, currency,
		       last_modified, modified_by_id, modified_by_type, status
		FROM otc_negotiations WHERE id = $1`, id,
	).Scan(
		&n.Id, &n.Ticker, &n.SellerId, &n.SellerType, &n.BuyerId, &n.BuyerType,
		&n.Amount, &n.PricePerStock, &settlementDate, &n.Premium, &n.Currency,
		&lastModified, &modifiedByID, &modifiedByType, &n.Status,
	)
	if err == sql.ErrNoRows {
		return nil, status.Error(codes.NotFound, "negotiation not found")
	} else if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to fetch negotiation: %v", err)
	}

	n.SettlementDate = settlementDate
	n.LastModified = lastModified.Format(time.RFC3339)
	n.SellerName = getUserName(s.EmployeeDB, s.ClientDB, n.SellerId, n.SellerType)
	n.BuyerName = getUserName(s.EmployeeDB, s.ClientDB, n.BuyerId, n.BuyerType)

	if modifiedByID.Valid {
		n.ModifiedById = modifiedByID.Int64
	}
	if modifiedByType.Valid {
		n.ModifiedByType = modifiedByType.String
		n.ModifiedByName = getUserName(s.EmployeeDB, s.ClientDB, n.ModifiedById, n.ModifiedByType)
	}

	return &n, nil
}

package handlers

import (
	"context"
	"database/sql"

	"github.com/RAF-SI-2025/EXBanka-4-Backend/services/portfolio-service/repository"
	pb "github.com/RAF-SI-2025/EXBanka-4-Backend/shared/pb/portfolio"
	pb_sec "github.com/RAF-SI-2025/EXBanka-4-Backend/shared/pb/securities"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

// SecurityPriceFetcher is the subset of SecuritiesServiceClient we need.
type SecurityPriceFetcher interface {
	GetListingById(ctx context.Context, in *pb_sec.GetListingByIdRequest, opts ...grpc.CallOption) (*pb_sec.GetListingByIdResponse, error)
}

type PortfolioServer struct {
	pb.UnimplementedPortfolioServiceServer
	DB               *sql.DB
	SecuritiesClient SecurityPriceFetcher
}

func (s *PortfolioServer) UpdateHolding(ctx context.Context, req *pb.UpdateHoldingRequest) (*pb.UpdateHoldingResponse, error) {
	if req.Quantity <= 0 {
		return nil, status.Error(codes.InvalidArgument, "quantity must be positive")
	}
	if req.Direction != "BUY" && req.Direction != "SELL" {
		return nil, status.Error(codes.InvalidArgument, "direction must be BUY or SELL")
	}

	err := repository.UpsertHolding(ctx, s.DB, req.UserId, req.UserType, req.ListingId, req.AccountId, req.Quantity, req.Price, req.Direction)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "upsert holding: %v", err)
	}
	return &pb.UpdateHoldingResponse{}, nil
}

func userTypeFromCtx(ctx context.Context) string {
	if md, ok := metadata.FromIncomingContext(ctx); ok {
		if vals := md.Get("user-type"); len(vals) > 0 {
			return vals[0]
		}
	}
	return ""
}

func (s *PortfolioServer) GetPortfolio(ctx context.Context, req *pb.GetPortfolioRequest) (*pb.GetPortfolioResponse, error) {
	entries, err := repository.GetHoldings(ctx, s.DB, req.UserId, userTypeFromCtx(ctx))
	if err != nil {
		return nil, status.Errorf(codes.Internal, "get holdings: %v", err)
	}

	pbEntries := make([]*pb.PortfolioEntry, 0, len(entries))
	for _, e := range entries {
		entry := &pb.PortfolioEntry{
			Id:           e.ID,
			ListingId:    e.ListingID,
			Amount:       e.Amount,
			BuyPrice:     e.BuyPrice,
			LastModified: e.LastModified.Format("2006-01-02T15:04:05"),
			IsPublic:     e.IsPublic,
			PublicAmount: e.PublicAmount,
			AccountId:    e.AccountID,
		}

		if s.SecuritiesClient != nil {
			resp, secErr := s.SecuritiesClient.GetListingById(ctx, &pb_sec.GetListingByIdRequest{Id: e.ListingID})
			if secErr == nil && resp.Summary != nil {
				entry.Ticker = resp.Summary.Ticker
				entry.AssetType = resp.Summary.Type
				entry.Price = resp.Summary.Price
				entry.Profit = (resp.Summary.Price - e.BuyPrice) * float64(e.Amount)
			}
		}

		pbEntries = append(pbEntries, entry)
	}
	return &pb.GetPortfolioResponse{Entries: pbEntries}, nil
}

func (s *PortfolioServer) GetProfit(ctx context.Context, req *pb.GetProfitRequest) (*pb.GetProfitResponse, error) {
	entries, err := repository.GetHoldings(ctx, s.DB, req.UserId, userTypeFromCtx(ctx))
	if err != nil {
		return nil, status.Errorf(codes.Internal, "get holdings: %v", err)
	}

	var totalProfit float64
	if s.SecuritiesClient != nil {
		for _, e := range entries {
			resp, secErr := s.SecuritiesClient.GetListingById(ctx, &pb_sec.GetListingByIdRequest{Id: e.ListingID})
			if secErr == nil && resp.Summary != nil {
				totalProfit += (resp.Summary.Price - e.BuyPrice) * float64(e.Amount)
			}
		}
	}

	return &pb.GetProfitResponse{TotalProfit: totalProfit}, nil
}

func (s *PortfolioServer) SetPublicAmount(_ context.Context, _ *pb.SetPublicAmountRequest) (*pb.SetPublicAmountResponse, error) {
	return nil, status.Error(codes.Unimplemented, "implemented in issue #147")
}

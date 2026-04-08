package handlers

import (
	"context"
	"database/sql"

	pb "github.com/RAF-SI-2025/EXBanka-4-Backend/shared/pb/order"
)

type OrderServer struct {
	pb.UnimplementedOrderServiceServer
	DB *sql.DB
}

func (s *OrderServer) Ping(ctx context.Context, req *pb.PingRequest) (*pb.PingResponse, error) {
	return &pb.PingResponse{Message: "order-service OK"}, nil
}

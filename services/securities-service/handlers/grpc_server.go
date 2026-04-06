package handlers

import (
	"context"
	"database/sql"

	pb "github.com/RAF-SI-2025/EXBanka-4-Backend/shared/pb/securities"
)

type SecuritiesServer struct {
	pb.UnimplementedSecuritiesServiceServer
	DB *sql.DB
}

func (s *SecuritiesServer) Ping(_ context.Context, _ *pb.PingRequest) (*pb.PingResponse, error) {
	return &pb.PingResponse{Message: "securities-service ok"}, nil
}

package main

import (
	"log"
	"net"
	"os"

	portfoliodb "github.com/RAF-SI-2025/EXBanka-4-Backend/services/portfolio-service/db"
	"github.com/RAF-SI-2025/EXBanka-4-Backend/services/portfolio-service/handlers"
	pb "github.com/RAF-SI-2025/EXBanka-4-Backend/shared/pb/portfolio"
	pb_sec "github.com/RAF-SI-2025/EXBanka-4-Backend/shared/pb/securities"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

const grpcPort = ":50062"

func main() {
	db, err := portfoliodb.Connect(os.Getenv("PORTFOLIO_DB_URL"))
	if err != nil {
		log.Fatalf("failed to connect to portfolio_db: %v", err)
	}
	defer func() { _ = db.Close() }()

	secConn, err := grpc.NewClient(os.Getenv("SECURITIES_SERVICE_ADDR"), grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("failed to connect to securities-service: %v", err)
	}
	defer func() { _ = secConn.Close() }()

	securitiesClient := pb_sec.NewSecuritiesServiceClient(secConn)

	lis, err := net.Listen("tcp", grpcPort)
	if err != nil {
		log.Fatalf("failed to listen on %s: %v", grpcPort, err)
	}

	srv := grpc.NewServer()
	pb.RegisterPortfolioServiceServer(srv, &handlers.PortfolioServer{
		DB:               db,
		SecuritiesClient: securitiesClient,
	})

	log.Printf("portfolio-service gRPC server listening on %s", grpcPort)
	if err := srv.Serve(lis); err != nil {
		log.Fatalf("gRPC serve error: %v", err)
	}
}

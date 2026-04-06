package main

import (
	"log"
	"net"
	"os"

	secdb "github.com/RAF-SI-2025/EXBanka-4-Backend/services/securities-service/db"
	"github.com/RAF-SI-2025/EXBanka-4-Backend/services/securities-service/handlers"
	pb "github.com/RAF-SI-2025/EXBanka-4-Backend/shared/pb/securities"
	"google.golang.org/grpc"
)

const grpcPort = ":50060"

func main() {
	securitiesDB, err := secdb.Connect(os.Getenv("SECURITIES_DB_URL"))
	if err != nil {
		log.Fatalf("failed to connect to securities_db: %v", err)
	}
	defer securitiesDB.Close()

	lis, err := net.Listen("tcp", grpcPort)
	if err != nil {
		log.Fatalf("failed to listen on %s: %v", grpcPort, err)
	}

	srv := grpc.NewServer()
	pb.RegisterSecuritiesServiceServer(srv, &handlers.SecuritiesServer{
		DB: securitiesDB,
	})

	log.Printf("securities-service gRPC server listening on %s", grpcPort)
	if err := srv.Serve(lis); err != nil {
		log.Fatalf("gRPC serve error: %v", err)
	}
}

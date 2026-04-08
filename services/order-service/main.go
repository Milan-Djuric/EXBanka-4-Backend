package main

import (
	"log"
	"net"
	"os"

	orderdb "github.com/RAF-SI-2025/EXBanka-4-Backend/services/order-service/db"
	"github.com/RAF-SI-2025/EXBanka-4-Backend/services/order-service/handlers"
	pb "github.com/RAF-SI-2025/EXBanka-4-Backend/shared/pb/order"
	"google.golang.org/grpc"
)

const grpcPort = ":50061"

func main() {
	orderDB, err := orderdb.Connect(os.Getenv("ORDER_DB_URL"))
	if err != nil {
		log.Fatalf("failed to connect to order_db: %v", err)
	}
	defer orderDB.Close()

	lis, err := net.Listen("tcp", grpcPort)
	if err != nil {
		log.Fatalf("failed to listen on %s: %v", grpcPort, err)
	}

	srv := grpc.NewServer()
	pb.RegisterOrderServiceServer(srv, &handlers.OrderServer{
		DB: orderDB,
	})

	log.Printf("order-service gRPC server listening on %s", grpcPort)
	if err := srv.Serve(lis); err != nil {
		log.Fatalf("gRPC serve error: %v", err)
	}
}

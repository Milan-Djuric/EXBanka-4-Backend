package main

import (
	"log"
	"net"
	"os"

	otcdb "github.com/RAF-SI-2025/EXBanka-4-Backend/services/otc-service/db"
	"github.com/RAF-SI-2025/EXBanka-4-Backend/services/otc-service/handlers"
	pb "github.com/RAF-SI-2025/EXBanka-4-Backend/shared/pb/otc"
	"google.golang.org/grpc"
)

const grpcPort = ":50063"

func main() {
	otcDB, err := otcdb.Connect(os.Getenv("OTC_DB_URL"))
	if err != nil {
		log.Fatalf("failed to connect to otc_db: %v", err)
	}
	defer func() { _ = otcDB.Close() }()

	employeeDB, err := otcdb.Connect(os.Getenv("EMPLOYEE_DB_URL"))
	if err != nil {
		log.Fatalf("failed to connect to employee_db: %v", err)
	}
	defer func() { _ = employeeDB.Close() }()

	clientDB, err := otcdb.Connect(os.Getenv("CLIENT_DB_URL"))
	if err != nil {
		log.Fatalf("failed to connect to client_db: %v", err)
	}
	defer func() { _ = clientDB.Close() }()

	lis, err := net.Listen("tcp", grpcPort)
	if err != nil {
		log.Fatalf("failed to listen on %s: %v", grpcPort, err)
	}

	srv := grpc.NewServer()
	pb.RegisterOtcServiceServer(srv, &handlers.OtcServer{
		DB:         otcDB,
		EmployeeDB: employeeDB,
		ClientDB:   clientDB,
	})

	log.Printf("otc-service gRPC server listening on %s", grpcPort)
	if err := srv.Serve(lis); err != nil {
		log.Fatalf("gRPC serve error: %v", err)
	}
}

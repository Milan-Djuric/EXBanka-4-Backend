package main

import (
	"log"
	"net"
	"os"

	"google.golang.org/grpc"
	empdb "github.com/RAF-SI-2025/EXBanka-4-Backend/services/employee-service/db"
	"github.com/RAF-SI-2025/EXBanka-4-Backend/services/employee-service/handlers"
	pb "github.com/RAF-SI-2025/EXBanka-4-Backend/shared/pb/employee"
)

func main() {
	database, err := empdb.Connect(os.Getenv("DB_URL"))
	if err != nil {
		log.Fatalf("failed to connect to database: %v", err)
	}
	defer database.Close()

	lis, err := net.Listen("tcp", ":50051")
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	s := grpc.NewServer()
	srv := &handlers.EmployeeServer{DB: database}
	pb.RegisterEmployeeServiceServer(s, srv)
	srv.StartCronJobs()
	log.Println("employee-service listening on :50051")
	if err := s.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}

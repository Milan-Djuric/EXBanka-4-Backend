package handlers_test

import (
	"context"
	"testing"

	"github.com/RAF-SI-2025/EXBanka-4-Backend/services/order-service/handlers"
	pb "github.com/RAF-SI-2025/EXBanka-4-Backend/shared/pb/order"
	"github.com/stretchr/testify/assert"
)

func TestPing(t *testing.T) {
	srv := &handlers.OrderServer{}
	resp, err := srv.Ping(context.Background(), &pb.PingRequest{})
	assert.NoError(t, err)
	assert.Equal(t, "order-service OK", resp.Message)
}

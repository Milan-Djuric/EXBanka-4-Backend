package handlers

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	pb "github.com/RAF-SI-2025/EXBanka-4-Backend/shared/pb/securities"
)

// Ping godoc
// @Summary      Health check for securities-service
// @Tags         securities
// @Produce      json
// @Success      200  {object}  map[string]string
// @Router       /stock-exchanges/ping [get]
func PingSecurities(client pb.SecuritiesServiceClient) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
		defer cancel()

		resp, err := client.Ping(ctx, &pb.PingRequest{})
		if err != nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{"error": "securities-service unavailable"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"message": resp.Message})
	}
}

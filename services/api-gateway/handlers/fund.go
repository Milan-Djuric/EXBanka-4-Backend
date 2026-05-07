package handlers

import (
	"context"
	"net/http"
	"strconv"
	"time"

	"github.com/RAF-SI-2025/EXBanka-4-Backend/services/api-gateway/middleware"
	pb "github.com/RAF-SI-2025/EXBanka-4-Backend/shared/pb/fund"
	"github.com/gin-gonic/gin"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func fundToJSON(f *pb.FundResponse) gin.H {
	return gin.H{
		"id":                  f.Id,
		"name":                f.Name,
		"description":         f.Description,
		"minimumContribution": f.MinimumContribution,
		"managerId":           f.ManagerId,
		"managerName":         f.ManagerName,
		"liquidAssets":        f.LiquidAssets,
		"fundValue":           f.FundValue,
		"profit":              f.Profit,
		"accountNumber":       f.AccountNumber,
		"createdAt":           f.CreatedAt,
		"active":              f.Active,
		"accountId":           f.AccountId,
	}
}

func mapFundError(c *gin.Context, err error) {
	switch status.Code(err) {
	case codes.PermissionDenied:
		c.JSON(http.StatusForbidden, gin.H{"error": status.Convert(err).Message()})
	case codes.NotFound:
		c.JSON(http.StatusNotFound, gin.H{"error": status.Convert(err).Message()})
	case codes.AlreadyExists:
		c.JSON(http.StatusConflict, gin.H{"error": status.Convert(err).Message()})
	case codes.InvalidArgument:
		c.JSON(http.StatusBadRequest, gin.H{"error": status.Convert(err).Message()})
	case codes.FailedPrecondition:
		c.JSON(http.StatusUnprocessableEntity, gin.H{"error": status.Convert(err).Message()})
	default:
		c.JSON(http.StatusInternalServerError, gin.H{"error": status.Convert(err).Message()})
	}
}

// CreateFund godoc
// @Summary      Create an investment fund (SUPERVISOR)
// @Tags         investment-funds
// @Accept       json
// @Produce      json
// @Success      201  {object}  map[string]interface{}
// @Router       /investment/funds [post]
func CreateFund(client pb.FundServiceClient) gin.HandlerFunc {
	return func(c *gin.Context) {
		employeeID, err := middleware.GetUserIDFromToken(c)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "could not extract identity from token"})
			return
		}

		var req struct {
			Name                string  `json:"name"                binding:"required"`
			Description         string  `json:"description"`
			MinimumContribution float64 `json:"minimumContribution" binding:"required"`
			ManagerId           int64   `json:"managerId"           binding:"required"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		ctx, cancel := context.WithTimeout(c.Request.Context(), 15*time.Second)
		defer cancel()

		resp, err := client.CreateFund(ctx, &pb.CreateFundRequest{
			Name:                req.Name,
			Description:         req.Description,
			MinimumContribution: req.MinimumContribution,
			ManagerId:           req.ManagerId,
			CreatedById:         employeeID,
		})
		if err != nil {
			mapFundError(c, err)
			return
		}
		c.JSON(http.StatusCreated, fundToJSON(resp))
	}
}

// ListFunds godoc
// @Summary      List investment funds
// @Tags         investment-funds
// @Produce      json
// @Param        managerId  query  int  false  "Filter by manager ID"
// @Success      200  {array}  map[string]interface{}
// @Router       /investment/funds [get]
func ListFunds(client pb.FundServiceClient) gin.HandlerFunc {
	return func(c *gin.Context) {
		if _, err := middleware.GetUserIDFromToken(c); err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "could not extract identity from token"})
			return
		}

		var managerIDFilter int64
		if v := c.Query("managerId"); v != "" {
			managerIDFilter, _ = strconv.ParseInt(v, 10, 64)
		}

		ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
		defer cancel()

		resp, err := client.ListFunds(ctx, &pb.ListFundsRequest{ManagerIdFilter: managerIDFilter})
		if err != nil {
			mapFundError(c, err)
			return
		}

		result := make([]gin.H, 0, len(resp.Funds))
		for _, f := range resp.Funds {
			result = append(result, fundToJSON(f))
		}
		c.JSON(http.StatusOK, result)
	}
}

// GetFund godoc
// @Summary      Get a single investment fund
// @Tags         investment-funds
// @Produce      json
// @Param        id  path  int  true  "Fund ID"
// @Success      200  {object}  map[string]interface{}
// @Router       /investment/funds/{id} [get]
func GetFund(client pb.FundServiceClient) gin.HandlerFunc {
	return func(c *gin.Context) {
		if _, err := middleware.GetUserIDFromToken(c); err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "could not extract identity from token"})
			return
		}

		fundID, err := strconv.ParseInt(c.Param("id"), 10, 64)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid fund id"})
			return
		}

		ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
		defer cancel()

		resp, err := client.GetFund(ctx, &pb.GetFundRequest{Id: fundID})
		if err != nil {
			mapFundError(c, err)
			return
		}
		c.JSON(http.StatusOK, fundToJSON(resp))
	}
}

// UpdateFund godoc
// @Summary      Update an investment fund (SUPERVISOR)
// @Tags         investment-funds
// @Accept       json
// @Produce      json
// @Param        id  path  int  true  "Fund ID"
// @Success      200  {object}  map[string]interface{}
// @Router       /investment/funds/{id} [put]
func UpdateFund(client pb.FundServiceClient) gin.HandlerFunc {
	return func(c *gin.Context) {
		if _, err := middleware.GetUserIDFromToken(c); err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "could not extract identity from token"})
			return
		}

		fundID, err := strconv.ParseInt(c.Param("id"), 10, 64)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid fund id"})
			return
		}

		var req struct {
			Name                string  `json:"name"`
			Description         string  `json:"description"`
			MinimumContribution float64 `json:"minimumContribution"`
			ManagerId           int64   `json:"managerId"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
		defer cancel()

		resp, err := client.UpdateFund(ctx, &pb.UpdateFundRequest{
			Id:                  fundID,
			Name:                req.Name,
			Description:         req.Description,
			MinimumContribution: req.MinimumContribution,
			ManagerId:           req.ManagerId,
		})
		if err != nil {
			mapFundError(c, err)
			return
		}
		c.JSON(http.StatusOK, fundToJSON(resp))
	}
}

// DeleteFund godoc
// @Summary      Delete an investment fund (SUPERVISOR)
// @Tags         investment-funds
// @Produce      json
// @Param        id  path  int  true  "Fund ID"
// @Success      200  {object}  map[string]interface{}
// @Router       /investment/funds/{id} [delete]
func DeleteFund(client pb.FundServiceClient) gin.HandlerFunc {
	return func(c *gin.Context) {
		if _, err := middleware.GetUserIDFromToken(c); err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "could not extract identity from token"})
			return
		}

		fundID, err := strconv.ParseInt(c.Param("id"), 10, 64)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid fund id"})
			return
		}

		ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
		defer cancel()

		_, err = client.DeleteFund(ctx, &pb.DeleteFundRequest{Id: fundID})
		if err != nil {
			mapFundError(c, err)
			return
		}
		c.JSON(http.StatusOK, gin.H{"message": "fund deleted"})
	}
}

func InvestFund(client pb.FundServiceClient) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, err := middleware.GetUserIDFromToken(c)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "could not extract identity from token"})
			return
		}

		callerRole := middleware.GetCallerRoleFromToken(c)
		clientType := "CLIENT"
		if callerRole == "EMPLOYEE" {
			clientType = "BANK"
		}

		fundID, err := strconv.ParseInt(c.Param("id"), 10, 64)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid fund id"})
			return
		}

		var req struct {
			SourceAccountId int64   `json:"sourceAccountId" binding:"required"`
			Amount          float64 `json:"amount"          binding:"required"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		ctx, cancel := context.WithTimeout(c.Request.Context(), 15*time.Second)
		defer cancel()

		resp, err := client.InvestFund(ctx, &pb.InvestFundRequest{
			FundId:          fundID,
			ClientId:        userID,
			ClientType:      clientType,
			SourceAccountId: req.SourceAccountId,
			Amount:          req.Amount,
		})
		if err != nil {
			mapFundError(c, err)
			return
		}
		c.JSON(http.StatusOK, fundToJSON(resp))
	}
}

func WithdrawFund(client pb.FundServiceClient) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, err := middleware.GetUserIDFromToken(c)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "could not extract identity from token"})
			return
		}

		callerRole := middleware.GetCallerRoleFromToken(c)
		clientType := "CLIENT"
		if callerRole == "EMPLOYEE" {
			clientType = "BANK"
		}

		fundID, err := strconv.ParseInt(c.Param("id"), 10, 64)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid fund id"})
			return
		}

		var req struct {
			DestinationAccountId int64   `json:"destinationAccountId" binding:"required"`
			Amount               float64 `json:"amount"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		ctx, cancel := context.WithTimeout(c.Request.Context(), 15*time.Second)
		defer cancel()

		resp, err := client.WithdrawFund(ctx, &pb.WithdrawFundRequest{
			FundId:               fundID,
			ClientId:             userID,
			ClientType:           clientType,
			DestinationAccountId: req.DestinationAccountId,
			Amount:               req.Amount,
		})
		if err != nil {
			mapFundError(c, err)
			return
		}
		c.JSON(http.StatusOK, fundToJSON(resp))
	}
}

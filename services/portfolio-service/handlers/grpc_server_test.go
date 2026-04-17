package handlers

import (
	"context"
	"testing"
	"time"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	pb "github.com/RAF-SI-2025/EXBanka-4-Backend/shared/pb/portfolio"
	pb_sec "github.com/RAF-SI-2025/EXBanka-4-Backend/shared/pb/securities"
	"google.golang.org/grpc"
)

// mockSecClient implements SecurityPriceFetcher for tests.
type mockSecClient struct {
	price     float64
	ticker    string
	assetType string
}

func (m *mockSecClient) GetListingById(_ context.Context, _ *pb_sec.GetListingByIdRequest, _ ...grpc.CallOption) (*pb_sec.GetListingByIdResponse, error) {
	return &pb_sec.GetListingByIdResponse{
		Summary: &pb_sec.ListingSummary{
			Price:  m.price,
			Ticker: m.ticker,
			Type:   m.assetType,
		},
	}, nil
}

func newServer(t *testing.T) (*PortfolioServer, sqlmock.Sqlmock) {
	t.Helper()
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	return &PortfolioServer{DB: db}, mock
}

func newServerWithSec(t *testing.T, sec SecurityPriceFetcher) (*PortfolioServer, sqlmock.Sqlmock) {
	t.Helper()
	srv, mock := newServer(t)
	srv.SecuritiesClient = sec
	return srv, mock
}

// ── UpdateHolding ─────────────────────────────────────────────────────────────

func TestUpdateHolding_Buy_NewEntry(t *testing.T) {
	srv, mock := newServer(t)

	mock.ExpectExec(`INSERT INTO portfolio_entry`).
		WithArgs(int64(1), "CLIENT", int64(10), int32(5), float64(100.0), int64(42)).
		WillReturnResult(sqlmock.NewResult(1, 1))

	_, err := srv.UpdateHolding(context.Background(), &pb.UpdateHoldingRequest{
		UserId:    1,
		UserType:  "CLIENT",
		ListingId: 10,
		Quantity:  5,
		Price:     100.0,
		Direction: "BUY",
		AccountId: 42,
	})
	require.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestUpdateHolding_Buy_ExistingEntry_WeightedAvg(t *testing.T) {
	srv, mock := newServer(t)

	// ON CONFLICT DO UPDATE — same INSERT statement handles both cases
	mock.ExpectExec(`INSERT INTO portfolio_entry`).
		WithArgs(int64(2), "EMPLOYEE", int64(20), int32(3), float64(200.0), int64(99)).
		WillReturnResult(sqlmock.NewResult(0, 1))

	_, err := srv.UpdateHolding(context.Background(), &pb.UpdateHoldingRequest{
		UserId:    2,
		UserType:  "EMPLOYEE",
		ListingId: 20,
		Quantity:  3,
		Price:     200.0,
		Direction: "BUY",
		AccountId: 99,
	})
	require.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestUpdateHolding_Sell_Partial(t *testing.T) {
	srv, mock := newServer(t)

	mock.ExpectExec(`UPDATE portfolio_entry`).
		WithArgs(int32(2), int64(1), int64(10)).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(`DELETE FROM portfolio_entry`).
		WithArgs(int64(1), int64(10)).
		WillReturnResult(sqlmock.NewResult(0, 0))

	_, err := srv.UpdateHolding(context.Background(), &pb.UpdateHoldingRequest{
		UserId:    1,
		ListingId: 10,
		Quantity:  2,
		Price:     150.0,
		Direction: "SELL",
	})
	require.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestUpdateHolding_Sell_Full(t *testing.T) {
	srv, mock := newServer(t)

	mock.ExpectExec(`UPDATE portfolio_entry`).
		WithArgs(int32(5), int64(1), int64(10)).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(`DELETE FROM portfolio_entry`).
		WithArgs(int64(1), int64(10)).
		WillReturnResult(sqlmock.NewResult(0, 1)) // row deleted (amount hit 0)

	_, err := srv.UpdateHolding(context.Background(), &pb.UpdateHoldingRequest{
		UserId:    1,
		ListingId: 10,
		Quantity:  5,
		Price:     150.0,
		Direction: "SELL",
	})
	require.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestUpdateHolding_InvalidDirection(t *testing.T) {
	srv, _ := newServer(t)

	_, err := srv.UpdateHolding(context.Background(), &pb.UpdateHoldingRequest{
		UserId:    1,
		ListingId: 10,
		Quantity:  1,
		Direction: "HOLD",
	})
	require.Error(t, err)
}

func TestUpdateHolding_ZeroQuantity(t *testing.T) {
	srv, _ := newServer(t)

	_, err := srv.UpdateHolding(context.Background(), &pb.UpdateHoldingRequest{
		UserId:    1,
		ListingId: 10,
		Quantity:  0,
		Direction: "BUY",
	})
	require.Error(t, err)
}

// ── GetPortfolio ──────────────────────────────────────────────────────────────

func TestGetPortfolio_WithPriceEnrichment(t *testing.T) {
	sec := &mockSecClient{price: 200.0, ticker: "AAPL", assetType: "STOCK"}
	srv, mock := newServerWithSec(t, sec)

	rows := sqlmock.NewRows([]string{
		"id", "user_id", "user_type", "listing_id", "amount",
		"buy_price", "last_modified", "is_public", "public_amount", "account_id",
	}).AddRow(1, int64(1), "CLIENT", int64(10), int32(5), float64(150.0), time.Now(), false, 0, int64(42))

	mock.ExpectQuery(`SELECT`).WithArgs(int64(1)).WillReturnRows(rows)

	resp, err := srv.GetPortfolio(context.Background(), &pb.GetPortfolioRequest{UserId: 1})
	require.NoError(t, err)
	require.Len(t, resp.Entries, 1)

	e := resp.Entries[0]
	assert.Equal(t, "AAPL", e.Ticker)
	assert.Equal(t, "STOCK", e.AssetType)
	assert.InDelta(t, 200.0, e.Price, 0.001)
	// profit = (200 - 150) * 5 = 250
	assert.InDelta(t, 250.0, e.Profit, 0.001)
	assert.NoError(t, mock.ExpectationsWereMet())
}

// ── GetProfit ─────────────────────────────────────────────────────────────────

func TestGetProfit_HappyPath(t *testing.T) {
	// Two holdings: AAPL bought at 150 (5 shares, current 200) = +250
	//               MSFT bought at 300 (2 shares, current 280) = -40
	// total profit = 210
	sec := &mockSecClient{price: 200.0, ticker: "AAPL", assetType: "STOCK"}
	srv, mock := newServerWithSec(t, sec)

	now := time.Now()
	rows := sqlmock.NewRows([]string{
		"id", "user_id", "user_type", "listing_id", "amount",
		"buy_price", "last_modified", "is_public", "public_amount", "account_id",
	}).
		AddRow(1, int64(1), "CLIENT", int64(10), int32(5), float64(150.0), now, false, 0, int64(42)).
		AddRow(2, int64(1), "CLIENT", int64(20), int32(2), float64(300.0), now, false, 0, int64(42))

	mock.ExpectQuery(`SELECT`).WithArgs(int64(1)).WillReturnRows(rows)

	// Override mock to return different prices per call
	callCount := 0
	srv.SecuritiesClient = &callCountMockSecClient{
		prices: []float64{200.0, 280.0},
		call:   &callCount,
	}

	resp, err := srv.GetProfit(context.Background(), &pb.GetProfitRequest{UserId: 1})
	require.NoError(t, err)
	// (200-150)*5 + (280-300)*2 = 250 + (-40) = 210
	assert.InDelta(t, 210.0, resp.TotalProfit, 0.001)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetProfit_EmptyPortfolio(t *testing.T) {
	sec := &mockSecClient{}
	srv, mock := newServerWithSec(t, sec)

	rows := sqlmock.NewRows([]string{
		"id", "user_id", "user_type", "listing_id", "amount",
		"buy_price", "last_modified", "is_public", "public_amount", "account_id",
	})
	mock.ExpectQuery(`SELECT`).WithArgs(int64(99)).WillReturnRows(rows)

	resp, err := srv.GetProfit(context.Background(), &pb.GetProfitRequest{UserId: 99})
	require.NoError(t, err)
	assert.InDelta(t, 0.0, resp.TotalProfit, 0.001)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetProfit_NegativeProfit(t *testing.T) {
	sec := &mockSecClient{price: 80.0, ticker: "XYZ", assetType: "STOCK"}
	srv, mock := newServerWithSec(t, sec)

	rows := sqlmock.NewRows([]string{
		"id", "user_id", "user_type", "listing_id", "amount",
		"buy_price", "last_modified", "is_public", "public_amount", "account_id",
	}).AddRow(1, int64(1), "CLIENT", int64(5), int32(10), float64(100.0), time.Now(), false, 0, int64(42))

	mock.ExpectQuery(`SELECT`).WithArgs(int64(1)).WillReturnRows(rows)

	resp, err := srv.GetProfit(context.Background(), &pb.GetProfitRequest{UserId: 1})
	require.NoError(t, err)
	// (80 - 100) * 10 = -200
	assert.InDelta(t, -200.0, resp.TotalProfit, 0.001)
	assert.NoError(t, mock.ExpectationsWereMet())
}

// callCountMockSecClient returns different prices per sequential call.
type callCountMockSecClient struct {
	prices []float64
	call   *int
}

func (m *callCountMockSecClient) GetListingById(_ context.Context, _ *pb_sec.GetListingByIdRequest, _ ...grpc.CallOption) (*pb_sec.GetListingByIdResponse, error) {
	idx := *m.call
	*m.call++
	price := 0.0
	if idx < len(m.prices) {
		price = m.prices[idx]
	}
	return &pb_sec.GetListingByIdResponse{
		Summary: &pb_sec.ListingSummary{Price: price},
	}, nil
}

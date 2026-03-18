package handlers

import (
	"context"
	"fmt"
	"testing"
	"time"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	pb "github.com/RAF-SI-2025/EXBanka-4-Backend/shared/pb/payment"
)

// newMockServer returns a PaymentServer with two independent sqlmock DBs.
func newMockServer(t *testing.T) (*PaymentServer, sqlmock.Sqlmock, sqlmock.Sqlmock) {
	t.Helper()
	paymentDB, paymentMock, err := sqlmock.New()
	require.NoError(t, err)
	accountDB, accountMock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() {
		paymentDB.Close()
		accountDB.Close()
	})
	return &PaymentServer{DB: paymentDB, AccountDB: accountDB}, paymentMock, accountMock
}

// ---- GetPaymentById tests ----

func TestGetPaymentById_HappyPathWithRecipient(t *testing.T) {
	s, paymentMock, accountMock := newMockServer(t)

	ts := time.Date(2025, 6, 1, 10, 0, 0, 0, time.UTC)
	paymentMock.ExpectQuery("SELECT p.id, p.order_number").
		WithArgs(int64(1)).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "order_number", "from_account", "to_account",
			"initial_amount", "final_amount", "fee",
			"payment_code", "reference_number", "purpose",
			"timestamp", "status", "name",
		}).AddRow(1, "ORD-001", "ACC-100", "ACC-200", 300.0, 300.0, 0.0, "289", "RF01", "rent", ts, "COMPLETED", "Ana Petrović"))

	accountMock.ExpectQuery("SELECT owner_id FROM accounts").
		WithArgs("ACC-100").
		WillReturnRows(sqlmock.NewRows([]string{"owner_id"}).AddRow(int64(42)))

	resp, err := s.GetPaymentById(context.Background(), &pb.GetPaymentByIdRequest{PaymentId: 1, ClientId: 42})
	require.NoError(t, err)
	p := resp.Payment
	assert.Equal(t, int64(1), p.Id)
	assert.Equal(t, "Ana Petrović", p.RecipientName)
	assert.Equal(t, 300.0, p.InitialAmount)
	assert.Equal(t, ts.Format(time.RFC3339), p.Timestamp)
}

func TestGetPaymentById_HappyPathNoRecipient(t *testing.T) {
	s, paymentMock, accountMock := newMockServer(t)

	ts := time.Now()
	paymentMock.ExpectQuery("SELECT p.id, p.order_number").
		WithArgs(int64(2)).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "order_number", "from_account", "to_account",
			"initial_amount", "final_amount", "fee",
			"payment_code", "reference_number", "purpose",
			"timestamp", "status", "name",
		}).AddRow(2, "ORD-002", "ACC-100", "EXT-999", 150.0, 150.0, 0.0, "", "", "phone", ts, "COMPLETED", nil))

	accountMock.ExpectQuery("SELECT owner_id FROM accounts").
		WithArgs("ACC-100").
		WillReturnRows(sqlmock.NewRows([]string{"owner_id"}).AddRow(int64(42)))

	resp, err := s.GetPaymentById(context.Background(), &pb.GetPaymentByIdRequest{PaymentId: 2, ClientId: 42})
	require.NoError(t, err)
	assert.Equal(t, "", resp.Payment.RecipientName)
}

func TestGetPaymentById_NotFound(t *testing.T) {
	s, paymentMock, _ := newMockServer(t)

	paymentMock.ExpectQuery("SELECT p.id, p.order_number").
		WithArgs(int64(999)).
		WillReturnRows(sqlmock.NewRows([]string{}))

	_, err := s.GetPaymentById(context.Background(), &pb.GetPaymentByIdRequest{PaymentId: 999, ClientId: 1})
	require.Error(t, err)
	assert.Equal(t, codes.NotFound, status.Code(err))
}

func TestGetPaymentById_PermissionDenied(t *testing.T) {
	s, paymentMock, accountMock := newMockServer(t)

	ts := time.Now()
	paymentMock.ExpectQuery("SELECT p.id, p.order_number").
		WithArgs(int64(3)).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "order_number", "from_account", "to_account",
			"initial_amount", "final_amount", "fee",
			"payment_code", "reference_number", "purpose",
			"timestamp", "status", "name",
		}).AddRow(3, "ORD-003", "ACC-500", "ACC-600", 100.0, 100.0, 0.0, "", "", "", ts, "COMPLETED", nil))

	// account belongs to client 99, not 42
	accountMock.ExpectQuery("SELECT owner_id FROM accounts").
		WithArgs("ACC-500").
		WillReturnRows(sqlmock.NewRows([]string{"owner_id"}).AddRow(int64(99)))

	_, err := s.GetPaymentById(context.Background(), &pb.GetPaymentByIdRequest{PaymentId: 3, ClientId: 42})
	require.Error(t, err)
	assert.Equal(t, codes.PermissionDenied, status.Code(err))
}

func TestGetPaymentById_PaymentDBError(t *testing.T) {
	s, paymentMock, _ := newMockServer(t)

	paymentMock.ExpectQuery("SELECT p.id, p.order_number").
		WithArgs(int64(4)).
		WillReturnError(fmt.Errorf("db unavailable"))

	_, err := s.GetPaymentById(context.Background(), &pb.GetPaymentByIdRequest{PaymentId: 4, ClientId: 1})
	require.Error(t, err)
	assert.Equal(t, codes.Internal, status.Code(err))
}

// ---- GetPayments tests ----

func TestGetPayments_NoAccounts(t *testing.T) {
	s, _, accountMock := newMockServer(t)

	accountMock.ExpectQuery("SELECT account_number FROM accounts").
		WithArgs(int64(99)).
		WillReturnRows(sqlmock.NewRows([]string{"account_number"}))

	resp, err := s.GetPayments(context.Background(), &pb.GetPaymentsRequest{ClientId: 99})
	require.NoError(t, err)
	assert.Empty(t, resp.Payments)
	assert.NoError(t, accountMock.ExpectationsWereMet())
}

func TestGetPayments_AccountDBError(t *testing.T) {
	s, _, accountMock := newMockServer(t)

	accountMock.ExpectQuery("SELECT account_number FROM accounts").
		WithArgs(int64(1)).
		WillReturnError(fmt.Errorf("connection refused"))

	_, err := s.GetPayments(context.Background(), &pb.GetPaymentsRequest{ClientId: 1})
	require.Error(t, err)
	assert.Equal(t, codes.Internal, status.Code(err))
}

func TestGetPayments_NoFilters(t *testing.T) {
	s, paymentMock, accountMock := newMockServer(t)

	accountMock.ExpectQuery("SELECT account_number FROM accounts").
		WithArgs(int64(1)).
		WillReturnRows(sqlmock.NewRows([]string{"account_number"}).
			AddRow("ACC-001").AddRow("ACC-002"))

	ts := time.Date(2025, 6, 1, 10, 0, 0, 0, time.UTC)
	paymentMock.ExpectQuery("SELECT id, order_number").
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "order_number", "from_account", "to_account",
			"initial_amount", "final_amount", "fee",
			"payment_code", "reference_number", "purpose",
			"timestamp", "status",
		}).AddRow(1, "ORD-001", "ACC-001", "EXT-999", 500.0, 500.0, 0.0, "289", "RF001", "rent", ts, "COMPLETED"))

	resp, err := s.GetPayments(context.Background(), &pb.GetPaymentsRequest{ClientId: 1})
	require.NoError(t, err)
	require.Len(t, resp.Payments, 1)
	p := resp.Payments[0]
	assert.Equal(t, int64(1), p.Id)
	assert.Equal(t, "ORD-001", p.OrderNumber)
	assert.Equal(t, "ACC-001", p.FromAccount)
	assert.Equal(t, "EXT-999", p.ToAccount)
	assert.Equal(t, 500.0, p.InitialAmount)
	assert.Equal(t, "COMPLETED", p.Status)
	assert.Equal(t, ts.Format(time.RFC3339), p.Timestamp)

	assert.NoError(t, accountMock.ExpectationsWereMet())
	assert.NoError(t, paymentMock.ExpectationsWereMet())
}

func TestGetPayments_MultiplePayments(t *testing.T) {
	s, paymentMock, accountMock := newMockServer(t)

	accountMock.ExpectQuery("SELECT account_number FROM accounts").
		WithArgs(int64(2)).
		WillReturnRows(sqlmock.NewRows([]string{"account_number"}).AddRow("ACC-100"))

	ts1 := time.Date(2025, 5, 1, 0, 0, 0, 0, time.UTC)
	ts2 := time.Date(2025, 4, 1, 0, 0, 0, 0, time.UTC)
	rows := sqlmock.NewRows([]string{
		"id", "order_number", "from_account", "to_account",
		"initial_amount", "final_amount", "fee",
		"payment_code", "reference_number", "purpose",
		"timestamp", "status",
	}).
		AddRow(10, "ORD-010", "ACC-100", "EXT-111", 200.0, 200.0, 0.0, "", "", "electricity", ts1, "COMPLETED").
		AddRow(9, "ORD-009", "ACC-100", "EXT-222", 100.0, 99.5, 0.5, "", "", "phone", ts2, "COMPLETED")

	paymentMock.ExpectQuery("SELECT id, order_number").WillReturnRows(rows)

	resp, err := s.GetPayments(context.Background(), &pb.GetPaymentsRequest{ClientId: 2})
	require.NoError(t, err)
	assert.Len(t, resp.Payments, 2)
	assert.Equal(t, int64(10), resp.Payments[0].Id)
	assert.Equal(t, int64(9), resp.Payments[1].Id)
}

func TestGetPayments_FilterByStatus(t *testing.T) {
	s, paymentMock, accountMock := newMockServer(t)

	accountMock.ExpectQuery("SELECT account_number FROM accounts").
		WithArgs(int64(3)).
		WillReturnRows(sqlmock.NewRows([]string{"account_number"}).AddRow("ACC-300"))

	ts := time.Date(2025, 3, 15, 8, 0, 0, 0, time.UTC)
	paymentMock.ExpectQuery("SELECT id, order_number").
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "order_number", "from_account", "to_account",
			"initial_amount", "final_amount", "fee",
			"payment_code", "reference_number", "purpose",
			"timestamp", "status",
		}).AddRow(5, "ORD-005", "ACC-300", "EXT-400", 150.0, 150.0, 0.0, "", "", "", ts, "PROCESSING"))

	resp, err := s.GetPayments(context.Background(), &pb.GetPaymentsRequest{
		ClientId: 3,
		Status:   "PROCESSING",
	})
	require.NoError(t, err)
	require.Len(t, resp.Payments, 1)
	assert.Equal(t, "PROCESSING", resp.Payments[0].Status)
}

func TestGetPayments_FilterByDateRange(t *testing.T) {
	s, paymentMock, accountMock := newMockServer(t)

	accountMock.ExpectQuery("SELECT account_number FROM accounts").
		WithArgs(int64(4)).
		WillReturnRows(sqlmock.NewRows([]string{"account_number"}).AddRow("ACC-400"))

	ts := time.Date(2025, 2, 10, 12, 0, 0, 0, time.UTC)
	paymentMock.ExpectQuery("SELECT id, order_number").
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "order_number", "from_account", "to_account",
			"initial_amount", "final_amount", "fee",
			"payment_code", "reference_number", "purpose",
			"timestamp", "status",
		}).AddRow(7, "ORD-007", "ACC-400", "EXT-500", 300.0, 300.0, 0.0, "", "", "", ts, "COMPLETED"))

	resp, err := s.GetPayments(context.Background(), &pb.GetPaymentsRequest{
		ClientId: 4,
		DateFrom: "2025-01-01T00:00:00Z",
		DateTo:   "2025-03-01T00:00:00Z",
	})
	require.NoError(t, err)
	assert.Len(t, resp.Payments, 1)
}

func TestGetPayments_FilterByAmountRange(t *testing.T) {
	s, paymentMock, accountMock := newMockServer(t)

	accountMock.ExpectQuery("SELECT account_number FROM accounts").
		WithArgs(int64(5)).
		WillReturnRows(sqlmock.NewRows([]string{"account_number"}).AddRow("ACC-500"))

	ts := time.Now()
	paymentMock.ExpectQuery("SELECT id, order_number").
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "order_number", "from_account", "to_account",
			"initial_amount", "final_amount", "fee",
			"payment_code", "reference_number", "purpose",
			"timestamp", "status",
		}).AddRow(8, "ORD-008", "ACC-500", "EXT-600", 250.0, 250.0, 0.0, "", "", "", ts, "COMPLETED"))

	resp, err := s.GetPayments(context.Background(), &pb.GetPaymentsRequest{
		ClientId:  5,
		AmountMin: 100.0,
		AmountMax: 500.0,
	})
	require.NoError(t, err)
	assert.Len(t, resp.Payments, 1)
	assert.Equal(t, 250.0, resp.Payments[0].InitialAmount)
}

func TestGetPayments_EmptyResult(t *testing.T) {
	s, paymentMock, accountMock := newMockServer(t)

	accountMock.ExpectQuery("SELECT account_number FROM accounts").
		WithArgs(int64(6)).
		WillReturnRows(sqlmock.NewRows([]string{"account_number"}).AddRow("ACC-600"))

	paymentMock.ExpectQuery("SELECT id, order_number").
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "order_number", "from_account", "to_account",
			"initial_amount", "final_amount", "fee",
			"payment_code", "reference_number", "purpose",
			"timestamp", "status",
		}))

	resp, err := s.GetPayments(context.Background(), &pb.GetPaymentsRequest{ClientId: 6})
	require.NoError(t, err)
	assert.Empty(t, resp.Payments)
}

func TestGetPayments_PaymentDBError(t *testing.T) {
	s, paymentMock, accountMock := newMockServer(t)

	accountMock.ExpectQuery("SELECT account_number FROM accounts").
		WithArgs(int64(7)).
		WillReturnRows(sqlmock.NewRows([]string{"account_number"}).AddRow("ACC-700"))

	paymentMock.ExpectQuery("SELECT id, order_number").
		WillReturnError(fmt.Errorf("query timeout"))

	_, err := s.GetPayments(context.Background(), &pb.GetPaymentsRequest{ClientId: 7})
	require.Error(t, err)
	assert.Equal(t, codes.Internal, status.Code(err))
}

func TestGetPayments_ScanError(t *testing.T) {
	s, paymentMock, accountMock := newMockServer(t)

	accountMock.ExpectQuery("SELECT account_number FROM accounts").
		WithArgs(int64(8)).
		WillReturnRows(sqlmock.NewRows([]string{"account_number"}).AddRow("ACC-800"))

	// Return a row with only 1 column — Scan expects 12, so it will fail
	paymentMock.ExpectQuery("SELECT id, order_number").
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(1))

	_, err := s.GetPayments(context.Background(), &pb.GetPaymentsRequest{ClientId: 8})
	require.Error(t, err)
	assert.Equal(t, codes.Internal, status.Code(err))
}

func TestGetPayments_AllFilters(t *testing.T) {
	s, paymentMock, accountMock := newMockServer(t)

	accountMock.ExpectQuery("SELECT account_number FROM accounts").
		WithArgs(int64(9)).
		WillReturnRows(sqlmock.NewRows([]string{"account_number"}).AddRow("ACC-900"))

	ts := time.Date(2025, 7, 20, 0, 0, 0, 0, time.UTC)
	paymentMock.ExpectQuery("SELECT id, order_number").
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "order_number", "from_account", "to_account",
			"initial_amount", "final_amount", "fee",
			"payment_code", "reference_number", "purpose",
			"timestamp", "status",
		}).AddRow(20, "ORD-020", "ACC-900", "EXT-999", 400.0, 395.0, 5.0, "221", "RF020", "salary", ts, "COMPLETED"))

	resp, err := s.GetPayments(context.Background(), &pb.GetPaymentsRequest{
		ClientId:  9,
		DateFrom:  "2025-07-01T00:00:00Z",
		DateTo:    "2025-08-01T00:00:00Z",
		AmountMin: 100.0,
		AmountMax: 1000.0,
		Status:    "COMPLETED",
	})
	require.NoError(t, err)
	require.Len(t, resp.Payments, 1)
	p := resp.Payments[0]
	assert.Equal(t, int64(20), p.Id)
	assert.Equal(t, 400.0, p.InitialAmount)
	assert.Equal(t, 5.0, p.Fee)
	assert.Equal(t, "salary", p.Purpose)
}

package handlers

import (
	"context"
	"database/sql"
	"testing"
	"time"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	pb "github.com/RAF-SI-2025/EXBanka-4-Backend/shared/pb/otc"
)

// newTestServer creates an OtcServer with three mock DBs (main, employee, client).
func newTestServer(t *testing.T) (*OtcServer, sqlmock.Sqlmock, sqlmock.Sqlmock, sqlmock.Sqlmock) {
	t.Helper()
	db, mainMock, err := sqlmock.New()
	require.NoError(t, err)
	empDB, empMock, err := sqlmock.New()
	require.NoError(t, err)
	clientDB, clientMock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() {
		_ = db.Close()
		_ = empDB.Close()
		_ = clientDB.Close()
	})
	return &OtcServer{DB: db, EmployeeDB: empDB, ClientDB: clientDB}, mainMock, empMock, clientMock
}

// negotiationColumns returns the columns scanned by fetchNegotiationByID.
func negotiationColumns() []string {
	return []string{
		"id", "ticker", "seller_id", "seller_type", "buyer_id", "buyer_type",
		"amount", "price_per_stock", "settlement_date", "premium", "currency",
		"last_modified", "modified_by_id", "modified_by_type", "status",
	}
}

// addFetchNegotiationRows sets up the mock expectations for fetchNegotiationByID
// (the SELECT + two name lookups for seller and buyer, plus modifiedBy name).
func addFetchNegotiationRows(mainMock, empMock, clientMock sqlmock.Sqlmock,
	id, sellerID, buyerID int64, sellerType, buyerType, negotiationStatus string) {
	now := time.Now()
	mainMock.ExpectQuery("SELECT id, ticker").
		WillReturnRows(sqlmock.NewRows(negotiationColumns()).
			AddRow(id, "AAPL", sellerID, sellerType, buyerID, buyerType,
				int32(100), float64(150.0), "2026-06-01", float64(0), "RSD",
				now, sql.NullInt64{Int64: buyerID, Valid: true},
				sql.NullString{String: buyerType, Valid: true},
				negotiationStatus))
	// seller name lookup (EMPLOYEE → empDB)
	empMock.ExpectQuery("SELECT first_name").
		WillReturnRows(sqlmock.NewRows([]string{"name"}).AddRow("Jane Doe"))
	// buyer name lookup (CLIENT → clientDB)
	clientMock.ExpectQuery("SELECT first_name").
		WillReturnRows(sqlmock.NewRows([]string{"name"}).AddRow("John Smith"))
	// modifiedBy name lookup (CLIENT type)
	clientMock.ExpectQuery("SELECT first_name").
		WillReturnRows(sqlmock.NewRows([]string{"name"}).AddRow("John Smith"))
}

// ---- TestCreateNegotiation_Happy ----

func TestCreateNegotiation_Happy(t *testing.T) {
	s, mainMock, empMock, clientMock := newTestServer(t)

	// INSERT returns id=1
	mainMock.ExpectQuery("INSERT INTO otc_negotiations").
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(int64(1)))

	// fetchNegotiationByID for id=1
	addFetchNegotiationRows(mainMock, empMock, clientMock, 1, 10, 20, "EMPLOYEE", "CLIENT", "PENDING_SELLER")

	resp, err := s.CreateNegotiation(context.Background(), &pb.CreateNegotiationRequest{
		Ticker:         "AAPL",
		SellerId:       10,
		SellerType:     "EMPLOYEE",
		BuyerId:        20,
		BuyerType:      "CLIENT",
		Amount:         100,
		PricePerStock:  150.0,
		SettlementDate: "2026-06-01",
		Currency:       "RSD",
	})
	require.NoError(t, err)
	assert.Equal(t, "AAPL", resp.Ticker)
	assert.Equal(t, "PENDING_SELLER", resp.Status)
	assert.Equal(t, int64(10), resp.SellerId)
	assert.Equal(t, int64(20), resp.BuyerId)
}

// ---- TestCreateNegotiation_MissingTicker ----

func TestCreateNegotiation_MissingTicker(t *testing.T) {
	s, _, _, _ := newTestServer(t)

	_, err := s.CreateNegotiation(context.Background(), &pb.CreateNegotiationRequest{
		Ticker:        "",
		SellerId:      10,
		SellerType:    "EMPLOYEE",
		BuyerId:       20,
		BuyerType:     "CLIENT",
		Amount:        100,
		PricePerStock: 150.0,
		Currency:      "RSD",
	})
	require.Error(t, err)
	assert.Equal(t, codes.InvalidArgument, status.Code(err))
}

// ---- TestListNegotiations_Empty ----

func TestListNegotiations_Empty(t *testing.T) {
	s, mainMock, _, _ := newTestServer(t)

	mainMock.ExpectQuery("SELECT id FROM otc_negotiations").
		WillReturnRows(sqlmock.NewRows([]string{"id"}))

	resp, err := s.ListNegotiations(context.Background(), &pb.ListNegotiationsRequest{
		CallerId:   5,
		CallerType: "CLIENT",
	})
	require.NoError(t, err)
	assert.Len(t, resp.Negotiations, 0)
}

// ---- TestCounterOffer_NotYourTurn ----

func TestCounterOffer_NotYourTurn(t *testing.T) {
	s, mainMock, _, _ := newTestServer(t)

	// State: seller=10/EMPLOYEE, buyer=20/CLIENT, status=PENDING_BUYER
	mainMock.ExpectQuery("SELECT seller_id, seller_type, buyer_id, buyer_type, status").
		WillReturnRows(sqlmock.NewRows([]string{"seller_id", "seller_type", "buyer_id", "buyer_type", "status"}).
			AddRow(int64(10), "EMPLOYEE", int64(20), "CLIENT", "PENDING_BUYER"))

	// Caller is seller (id=10, type=EMPLOYEE) → NOT their turn (PENDING_BUYER)
	_, err := s.CounterOffer(context.Background(), &pb.CounterOfferRequest{
		NegotiationId:  1,
		CallerId:       10,
		CallerType:     "EMPLOYEE",
		Amount:         90,
		PricePerStock:  155.0,
		SettlementDate: "2026-06-15",
	})
	require.Error(t, err)
	assert.Equal(t, codes.AlreadyExists, status.Code(err))
}

// ---- TestCounterOffer_Happy ----

func TestCounterOffer_Happy(t *testing.T) {
	s, mainMock, empMock, clientMock := newTestServer(t)

	// State: seller=10/EMPLOYEE, buyer=20/CLIENT, status=PENDING_SELLER
	mainMock.ExpectQuery("SELECT seller_id, seller_type, buyer_id, buyer_type, status").
		WillReturnRows(sqlmock.NewRows([]string{"seller_id", "seller_type", "buyer_id", "buyer_type", "status"}).
			AddRow(int64(10), "EMPLOYEE", int64(20), "CLIENT", "PENDING_SELLER"))

	// UPDATE
	mainMock.ExpectExec("UPDATE otc_negotiations").
		WillReturnResult(sqlmock.NewResult(1, 1))

	// fetchNegotiationByID — status flipped to PENDING_BUYER
	addFetchNegotiationRows(mainMock, empMock, clientMock, 1, 10, 20, "EMPLOYEE", "CLIENT", "PENDING_BUYER")

	resp, err := s.CounterOffer(context.Background(), &pb.CounterOfferRequest{
		NegotiationId:  1,
		CallerId:       10,
		CallerType:     "EMPLOYEE",
		Amount:         90,
		PricePerStock:  155.0,
		SettlementDate: "2026-06-15",
	})
	require.NoError(t, err)
	assert.Equal(t, "PENDING_BUYER", resp.Status)
}

// ---- TestAcceptNegotiation_NotParticipant ----

func TestAcceptNegotiation_NotParticipant(t *testing.T) {
	s, mainMock, _, _ := newTestServer(t)

	// State: seller=10, buyer=20; caller=99 (not a participant)
	mainMock.ExpectQuery("SELECT seller_id, seller_type, buyer_id, buyer_type, status").
		WillReturnRows(sqlmock.NewRows([]string{"seller_id", "seller_type", "buyer_id", "buyer_type", "status"}).
			AddRow(int64(10), "EMPLOYEE", int64(20), "CLIENT", "PENDING_SELLER"))

	_, err := s.AcceptNegotiation(context.Background(), &pb.AcceptNegotiationRequest{
		NegotiationId: 1,
		CallerId:      99,
		CallerType:    "CLIENT",
	})
	require.Error(t, err)
	assert.Equal(t, codes.PermissionDenied, status.Code(err))
}

// ---- TestRejectNegotiation_TerminalState ----

func TestRejectNegotiation_TerminalState(t *testing.T) {
	s, mainMock, _, _ := newTestServer(t)

	// State: already ACCEPTED
	mainMock.ExpectQuery("SELECT seller_id, seller_type, buyer_id, buyer_type, status").
		WillReturnRows(sqlmock.NewRows([]string{"seller_id", "seller_type", "buyer_id", "buyer_type", "status"}).
			AddRow(int64(10), "EMPLOYEE", int64(20), "CLIENT", "ACCEPTED"))

	// Caller is the seller — they are a participant, but negotiation is terminal
	_, err := s.RejectNegotiation(context.Background(), &pb.RejectNegotiationRequest{
		NegotiationId: 1,
		CallerId:      10,
		CallerType:    "EMPLOYEE",
	})
	require.Error(t, err)
	assert.Equal(t, codes.FailedPrecondition, status.Code(err))
}

// ---- TestPing ----

func TestPing_OtcService(t *testing.T) {
	s, _, _, _ := newTestServer(t)
	resp, err := s.Ping(context.Background(), &pb.PingRequest{})
	require.NoError(t, err)
	assert.Equal(t, "otc-service ok", resp.Message)
}

// ---- TestGetUserName_ZeroID ----

func TestGetUserName_ZeroID(t *testing.T) {
	// userID == 0 returns "" immediately without querying any DB
	name := getUserName(nil, nil, 0, "CLIENT")
	assert.Equal(t, "", name)
}

// ---- TestCreateNegotiation validation ----

func TestCreateNegotiation_InvalidAmount(t *testing.T) {
	s, _, _, _ := newTestServer(t)
	_, err := s.CreateNegotiation(context.Background(), &pb.CreateNegotiationRequest{
		Ticker:        "AAPL",
		Amount:        0,
		PricePerStock: 100.0,
	})
	require.Error(t, err)
	assert.Equal(t, codes.InvalidArgument, status.Code(err))
}

func TestCreateNegotiation_InvalidPrice(t *testing.T) {
	s, _, _, _ := newTestServer(t)
	_, err := s.CreateNegotiation(context.Background(), &pb.CreateNegotiationRequest{
		Ticker:        "AAPL",
		Amount:        10,
		PricePerStock: 0,
	})
	require.Error(t, err)
	assert.Equal(t, codes.InvalidArgument, status.Code(err))
}

func TestCreateNegotiation_DBError(t *testing.T) {
	s, mainMock, _, _ := newTestServer(t)
	mainMock.ExpectQuery("INSERT INTO otc_negotiations").
		WillReturnError(sql.ErrConnDone)
	_, err := s.CreateNegotiation(context.Background(), &pb.CreateNegotiationRequest{
		Ticker:        "AAPL",
		Amount:        10,
		PricePerStock: 100.0,
		Currency:      "RSD",
	})
	require.Error(t, err)
	assert.Equal(t, codes.Internal, status.Code(err))
}

// ---- TestListNegotiations ----

func TestListNegotiations_WithResults(t *testing.T) {
	s, mainMock, empMock, clientMock := newTestServer(t)

	mainMock.ExpectQuery("SELECT id FROM otc_negotiations").
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(int64(1)).AddRow(int64(2)))

	// fetchNegotiationByID for id=1
	addFetchNegotiationRows(mainMock, empMock, clientMock, 1, 10, 20, "EMPLOYEE", "CLIENT", "PENDING_SELLER")
	// fetchNegotiationByID for id=2
	addFetchNegotiationRows(mainMock, empMock, clientMock, 2, 10, 20, "EMPLOYEE", "CLIENT", "PENDING_BUYER")

	resp, err := s.ListNegotiations(context.Background(), &pb.ListNegotiationsRequest{
		CallerId:   10,
		CallerType: "EMPLOYEE",
	})
	require.NoError(t, err)
	assert.Len(t, resp.Negotiations, 2)
}

func TestListNegotiations_DBError(t *testing.T) {
	s, mainMock, _, _ := newTestServer(t)
	mainMock.ExpectQuery("SELECT id FROM otc_negotiations").
		WillReturnError(sql.ErrConnDone)
	_, err := s.ListNegotiations(context.Background(), &pb.ListNegotiationsRequest{})
	require.Error(t, err)
	assert.Equal(t, codes.Internal, status.Code(err))
}

// ---- TestGetNegotiation ----

func TestGetNegotiation_Happy(t *testing.T) {
	s, mainMock, empMock, clientMock := newTestServer(t)
	addFetchNegotiationRows(mainMock, empMock, clientMock, 5, 10, 20, "EMPLOYEE", "CLIENT", "PENDING_SELLER")

	resp, err := s.GetNegotiation(context.Background(), &pb.GetNegotiationRequest{NegotiationId: 5})
	require.NoError(t, err)
	assert.Equal(t, int64(5), resp.Id)
	assert.Equal(t, "PENDING_SELLER", resp.Status)
}

func TestGetNegotiation_NotFound(t *testing.T) {
	s, mainMock, _, _ := newTestServer(t)
	mainMock.ExpectQuery("SELECT id, ticker").
		WillReturnRows(sqlmock.NewRows(negotiationColumns()))

	_, err := s.GetNegotiation(context.Background(), &pb.GetNegotiationRequest{NegotiationId: 999})
	require.Error(t, err)
	assert.Equal(t, codes.NotFound, status.Code(err))
}

// ---- TestCounterOffer additional branches ----

func TestCounterOffer_NotFound(t *testing.T) {
	s, mainMock, _, _ := newTestServer(t)
	mainMock.ExpectQuery("SELECT seller_id, seller_type, buyer_id, buyer_type, status").
		WillReturnRows(sqlmock.NewRows([]string{"seller_id", "seller_type", "buyer_id", "buyer_type", "status"}))

	_, err := s.CounterOffer(context.Background(), &pb.CounterOfferRequest{
		NegotiationId: 999, CallerId: 10, CallerType: "EMPLOYEE",
	})
	require.Error(t, err)
	assert.Equal(t, codes.NotFound, status.Code(err))
}

func TestCounterOffer_NotParticipant(t *testing.T) {
	s, mainMock, _, _ := newTestServer(t)
	mainMock.ExpectQuery("SELECT seller_id, seller_type, buyer_id, buyer_type, status").
		WillReturnRows(sqlmock.NewRows([]string{"seller_id", "seller_type", "buyer_id", "buyer_type", "status"}).
			AddRow(int64(10), "EMPLOYEE", int64(20), "CLIENT", "PENDING_SELLER"))

	_, err := s.CounterOffer(context.Background(), &pb.CounterOfferRequest{
		NegotiationId: 1, CallerId: 99, CallerType: "CLIENT",
	})
	require.Error(t, err)
	assert.Equal(t, codes.PermissionDenied, status.Code(err))
}

func TestCounterOffer_TerminalState(t *testing.T) {
	s, mainMock, _, _ := newTestServer(t)
	mainMock.ExpectQuery("SELECT seller_id, seller_type, buyer_id, buyer_type, status").
		WillReturnRows(sqlmock.NewRows([]string{"seller_id", "seller_type", "buyer_id", "buyer_type", "status"}).
			AddRow(int64(10), "EMPLOYEE", int64(20), "CLIENT", "ACCEPTED"))

	_, err := s.CounterOffer(context.Background(), &pb.CounterOfferRequest{
		NegotiationId: 1, CallerId: 10, CallerType: "EMPLOYEE",
	})
	require.Error(t, err)
	assert.Equal(t, codes.FailedPrecondition, status.Code(err))
}

// Buyer counter-offers when status=PENDING_BUYER → flips to PENDING_SELLER
func TestCounterOffer_BuyerTurn(t *testing.T) {
	s, mainMock, empMock, clientMock := newTestServer(t)

	mainMock.ExpectQuery("SELECT seller_id, seller_type, buyer_id, buyer_type, status").
		WillReturnRows(sqlmock.NewRows([]string{"seller_id", "seller_type", "buyer_id", "buyer_type", "status"}).
			AddRow(int64(10), "EMPLOYEE", int64(20), "CLIENT", "PENDING_BUYER"))

	mainMock.ExpectExec("UPDATE otc_negotiations").
		WillReturnResult(sqlmock.NewResult(1, 1))

	addFetchNegotiationRows(mainMock, empMock, clientMock, 1, 10, 20, "EMPLOYEE", "CLIENT", "PENDING_SELLER")

	resp, err := s.CounterOffer(context.Background(), &pb.CounterOfferRequest{
		NegotiationId: 1, CallerId: 20, CallerType: "CLIENT",
		Amount: 80, PricePerStock: 160.0, SettlementDate: "2026-07-01",
	})
	require.NoError(t, err)
	assert.Equal(t, "PENDING_SELLER", resp.Status)
}

// ---- TestAcceptNegotiation additional branches ----

func TestAcceptNegotiation_NotFound(t *testing.T) {
	s, mainMock, _, _ := newTestServer(t)
	mainMock.ExpectQuery("SELECT seller_id, seller_type, buyer_id, buyer_type, status").
		WillReturnRows(sqlmock.NewRows([]string{"seller_id", "seller_type", "buyer_id", "buyer_type", "status"}))

	_, err := s.AcceptNegotiation(context.Background(), &pb.AcceptNegotiationRequest{
		NegotiationId: 999, CallerId: 10, CallerType: "EMPLOYEE",
	})
	require.Error(t, err)
	assert.Equal(t, codes.NotFound, status.Code(err))
}

func TestAcceptNegotiation_TerminalState(t *testing.T) {
	s, mainMock, _, _ := newTestServer(t)
	mainMock.ExpectQuery("SELECT seller_id, seller_type, buyer_id, buyer_type, status").
		WillReturnRows(sqlmock.NewRows([]string{"seller_id", "seller_type", "buyer_id", "buyer_type", "status"}).
			AddRow(int64(10), "EMPLOYEE", int64(20), "CLIENT", "REJECTED"))

	_, err := s.AcceptNegotiation(context.Background(), &pb.AcceptNegotiationRequest{
		NegotiationId: 1, CallerId: 10, CallerType: "EMPLOYEE",
	})
	require.Error(t, err)
	assert.Equal(t, codes.FailedPrecondition, status.Code(err))
}

func TestAcceptNegotiation_NotYourTurn(t *testing.T) {
	s, mainMock, _, _ := newTestServer(t)
	// PENDING_BUYER but caller is seller — not their turn
	mainMock.ExpectQuery("SELECT seller_id, seller_type, buyer_id, buyer_type, status").
		WillReturnRows(sqlmock.NewRows([]string{"seller_id", "seller_type", "buyer_id", "buyer_type", "status"}).
			AddRow(int64(10), "EMPLOYEE", int64(20), "CLIENT", "PENDING_BUYER"))

	_, err := s.AcceptNegotiation(context.Background(), &pb.AcceptNegotiationRequest{
		NegotiationId: 1, CallerId: 10, CallerType: "EMPLOYEE",
	})
	require.Error(t, err)
	assert.Equal(t, codes.AlreadyExists, status.Code(err))
}

// Seller accepts when it's their turn (PENDING_SELLER)
func TestAcceptNegotiation_Happy(t *testing.T) {
	s, mainMock, empMock, clientMock := newTestServer(t)

	mainMock.ExpectQuery("SELECT seller_id, seller_type, buyer_id, buyer_type, status").
		WillReturnRows(sqlmock.NewRows([]string{"seller_id", "seller_type", "buyer_id", "buyer_type", "status"}).
			AddRow(int64(10), "EMPLOYEE", int64(20), "CLIENT", "PENDING_SELLER"))

	mainMock.ExpectExec("UPDATE otc_negotiations").
		WillReturnResult(sqlmock.NewResult(1, 1))

	addFetchNegotiationRows(mainMock, empMock, clientMock, 1, 10, 20, "EMPLOYEE", "CLIENT", "ACCEPTED")

	resp, err := s.AcceptNegotiation(context.Background(), &pb.AcceptNegotiationRequest{
		NegotiationId: 1, CallerId: 10, CallerType: "EMPLOYEE",
	})
	require.NoError(t, err)
	assert.Equal(t, "ACCEPTED", resp.Status)
}

// ---- TestRejectNegotiation additional branches ----

func TestRejectNegotiation_NotFound(t *testing.T) {
	s, mainMock, _, _ := newTestServer(t)
	mainMock.ExpectQuery("SELECT seller_id, seller_type, buyer_id, buyer_type, status").
		WillReturnRows(sqlmock.NewRows([]string{"seller_id", "seller_type", "buyer_id", "buyer_type", "status"}))

	_, err := s.RejectNegotiation(context.Background(), &pb.RejectNegotiationRequest{
		NegotiationId: 999, CallerId: 10, CallerType: "EMPLOYEE",
	})
	require.Error(t, err)
	assert.Equal(t, codes.NotFound, status.Code(err))
}

func TestRejectNegotiation_NotParticipant(t *testing.T) {
	s, mainMock, _, _ := newTestServer(t)
	mainMock.ExpectQuery("SELECT seller_id, seller_type, buyer_id, buyer_type, status").
		WillReturnRows(sqlmock.NewRows([]string{"seller_id", "seller_type", "buyer_id", "buyer_type", "status"}).
			AddRow(int64(10), "EMPLOYEE", int64(20), "CLIENT", "PENDING_SELLER"))

	_, err := s.RejectNegotiation(context.Background(), &pb.RejectNegotiationRequest{
		NegotiationId: 1, CallerId: 99, CallerType: "CLIENT",
	})
	require.Error(t, err)
	assert.Equal(t, codes.PermissionDenied, status.Code(err))
}

func TestRejectNegotiation_Happy(t *testing.T) {
	s, mainMock, empMock, clientMock := newTestServer(t)

	mainMock.ExpectQuery("SELECT seller_id, seller_type, buyer_id, buyer_type, status").
		WillReturnRows(sqlmock.NewRows([]string{"seller_id", "seller_type", "buyer_id", "buyer_type", "status"}).
			AddRow(int64(10), "EMPLOYEE", int64(20), "CLIENT", "PENDING_SELLER"))

	mainMock.ExpectExec("UPDATE otc_negotiations").
		WillReturnResult(sqlmock.NewResult(1, 1))

	addFetchNegotiationRows(mainMock, empMock, clientMock, 1, 10, 20, "EMPLOYEE", "CLIENT", "REJECTED")

	resp, err := s.RejectNegotiation(context.Background(), &pb.RejectNegotiationRequest{
		NegotiationId: 1, CallerId: 10, CallerType: "EMPLOYEE",
	})
	require.NoError(t, err)
	assert.Equal(t, "REJECTED", resp.Status)
}

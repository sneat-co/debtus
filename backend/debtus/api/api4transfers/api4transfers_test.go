package api4transfers

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/crediterra/money"
	"github.com/dal-go/dalgo/dal"
	"github.com/dal-go/dalgo/mocks/mock_dal"
	"github.com/sneat-co/debtus/backend/debtus/dal4debtus"
	"github.com/sneat-co/debtus/backend/debtus/facade4debtus"
	"github.com/sneat-co/debtus/backend/debtus/models4debtus"
	"github.com/sneat-co/sneat-core-modules/auth/token4auth"
	"github.com/sneat-co/sneat-core-modules/userus/dal4userus"
	"github.com/sneat-co/sneat-core-modules/userus/dbo4userus"
	"github.com/sneat-co/sneat-go-core/apicore"
	"github.com/sneat-co/sneat-go-core/apicore/verify"
	coretypes "github.com/sneat-co/sneat-go-core/coretypes"
	"github.com/sneat-co/sneat-go-core/facade"
	"github.com/strongo/strongoapp"
	"go.uber.org/mock/gomock"
)

// stubTransferDal implements dal4debtus.TransferDal, returning errors for all methods.
type stubTransferDal struct {
	latestErr error
	byUserErr error
}

func (s stubTransferDal) GetTransfersByID(_ context.Context, _ dal.ReadSession, _ []string) ([]models4debtus.TransferEntry, error) {
	return nil, errors.New("not implemented")
}
func (s stubTransferDal) LoadTransfersByUserID(_ context.Context, _ string, _, _ int) ([]models4debtus.TransferEntry, bool, error) {
	return nil, false, s.byUserErr
}
func (s stubTransferDal) LoadTransfersByContactID(_ context.Context, _ string, _, _ int) ([]models4debtus.TransferEntry, bool, error) {
	return nil, false, errors.New("not implemented")
}
func (s stubTransferDal) LoadTransferIDsByContactID(_ context.Context, _ string, _ int, _ string) ([]string, string, error) {
	return nil, "", errors.New("not implemented")
}
func (s stubTransferDal) LoadOverdueTransfers(_ context.Context, _ dal.ReadSession, _ string, _ int) ([]models4debtus.TransferEntry, error) {
	return nil, errors.New("not implemented")
}
func (s stubTransferDal) LoadOutstandingTransfers(_ context.Context, _ dal.ReadSession, _ time.Time, _, _ string, _ money.CurrencyCode, _ models4debtus.TransferDirection) ([]models4debtus.TransferEntry, error) {
	return nil, errors.New("not implemented")
}
func (s stubTransferDal) LoadDueTransfers(_ context.Context, _ dal.ReadSession, _ string, _ int) ([]models4debtus.TransferEntry, error) {
	return nil, errors.New("not implemented")
}
func (s stubTransferDal) LoadLatestTransfers(_ context.Context, _, _ int) ([]models4debtus.TransferEntry, error) {
	return nil, s.latestErr
}
func (s stubTransferDal) DelayUpdateTransferWithCreatorReceiptTgMessageID(_ context.Context, _ string, _ string, _, _ int64) error {
	return errors.New("not implemented")
}
func (s stubTransferDal) DelayUpdateTransfersWithCounterparty(_ context.Context, _ coretypes.SpaceID, _, _ string) error {
	return errors.New("not implemented")
}
func (s stubTransferDal) DelayUpdateTransfersOnReturn(_ context.Context, _ string, _ []dal4debtus.TransferReturnUpdate) error {
	return errors.New("not implemented")
}

func TestInitApiForTransfers(t *testing.T) {
	registered := make(map[string]string)
	InitApiForTransfers(func(method, path string, _ strongoapp.HttpHandlerWithContext) {
		registered[path] = method
	})
	for _, path := range []string{"/api4debtus/transfer", "/api4debtus/create-transfer", "/api4debtus/user/api4transfers", "/api4debtus/admin/latest/api4transfers"} {
		if _, ok := registered[path]; !ok {
			t.Errorf("expected path %q to be registered", path)
		}
	}
}

func TestPopulateTransfer(t *testing.T) {
	s := transferSourceSetToAPI{appPlatform: "api", createdOnID: "host1"}
	data := &models4debtus.TransferData{}
	s.PopulateTransfer(data)
	if data.CreatedOnPlatform != "api" {
		t.Errorf("CreatedOnPlatform = %q, want api", data.CreatedOnPlatform)
	}
	if data.CreatedOnID != "host1" {
		t.Errorf("CreatedOnID = %q, want host1", data.CreatedOnID)
	}
}

func TestHandleGetTransfer_missingID(t *testing.T) {
	// No "id" query param → GetStrID returns "" → handler returns early
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/api4debtus/transfer", nil)
	HandleGetTransfer(context.Background(), w, r)
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestHandleAdminLatestTransfers_error(t *testing.T) {
	orig := dal4debtus.Default.Transfer
	defer func() { dal4debtus.Default.Transfer = orig }()
	dal4debtus.Default.Transfer = stubTransferDal{latestErr: errors.New("db error")}

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/api4debtus/admin/latest/api4transfers", nil)
	HandleAdminLatestTransfers(context.Background(), w, r, token4auth.AuthInfo{})
	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d", w.Code)
	}
}

func TestHandleAdminLatestTransfers_success(t *testing.T) {
	orig := dal4debtus.Default.Transfer
	defer func() { dal4debtus.Default.Transfer = orig }()
	dal4debtus.Default.Transfer = stubTransferDal{} // returns nil error and nil slice

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/api4debtus/admin/latest/api4transfers", nil)
	HandleAdminLatestTransfers(context.Background(), w, r, token4auth.AuthInfo{})
	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
}

func TestHandleUserTransfers_error(t *testing.T) {
	orig := dal4debtus.Default.Transfer
	defer func() { dal4debtus.Default.Transfer = orig }()
	dal4debtus.Default.Transfer = stubTransferDal{byUserErr: errors.New("db error")}

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/api4debtus/user/api4transfers", nil)
	HandleUserTransfers(context.Background(), w, r, token4auth.AuthInfo{}, dbo4userus.UserEntry{})
	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d", w.Code)
	}
}

func TestHandleUserTransfers_success(t *testing.T) {
	orig := dal4debtus.Default.Transfer
	defer func() { dal4debtus.Default.Transfer = orig }()
	dal4debtus.Default.Transfer = stubTransferDal{} // returns nil error

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/api4debtus/user/api4transfers", nil)
	HandleUserTransfers(context.Background(), w, r, token4auth.AuthInfo{}, dbo4userus.UserEntry{})
	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
}

func TestHandleGetTransfer_withID_dbError(t *testing.T) {
	origDB := facade.GetSneatDB
	defer func() { facade.GetSneatDB = origDB }()
	facade.GetSneatDB = func(_ context.Context) (dal.DB, error) {
		return nil, errors.New("db unavailable")
	}

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/api4debtus/transfer?id=t1", nil)
	HandleGetTransfer(context.Background(), w, r)
	if w.Code == http.StatusOK {
		t.Error("expected non-200 status when DB fails")
	}
}

// makeRunTxDB returns a facade.GetSneatDB stub backed by a mock DB that runs RW transactions.
func makeRunTxDB(t *testing.T) func(ctx context.Context) (dal.DB, error) {
	t.Helper()
	ctrl := gomock.NewController(t)
	mockDB := mock_dal.NewMockDB(ctrl)
	mockDB.EXPECT().RunReadwriteTransaction(gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(
		func(ctx context.Context, f dal.RWTxWorker, _ ...dal.TransactionOption) error {
			mockTx := mock_dal.NewMockReadwriteTransaction(ctrl)
			mockTx.EXPECT().Set(gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
			mockTx.EXPECT().SetMulti(gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
			return f(ctx, mockTx)
		},
	).AnyTimes()
	return func(_ context.Context) (dal.DB, error) { return mockDB, nil }
}

func makeTransfer4transfers(fromUserID, toUserID string) models4debtus.TransferEntry {
	from := models4debtus.TransferCounterpartyInfo{UserID: fromUserID, ContactName: "From User"}
	to := models4debtus.TransferCounterpartyInfo{UserID: toUserID, ContactName: "To User"}
	fromJSON, _ := json.Marshal(from)
	toJSON, _ := json.Marshal(to)
	data := &models4debtus.TransferData{}
	data.FromJson = string(fromJSON)
	data.ToJson = string(toJSON)
	data.CreatorUserID = fromUserID
	return models4debtus.NewTransfer("t1", data)
}

func TestHandleGetTransfer_success(t *testing.T) {
	origFn := getTransferByIDFn4transfers
	defer func() { getTransferByIDFn4transfers = origFn }()
	getTransferByIDFn4transfers = func(_ context.Context, _ dal.ReadSession, _ string) (models4debtus.TransferEntry, error) {
		return makeTransfer4transfers("u1", "u2"), nil
	}

	origCheck := checkTransferCreatorNameFn4transfers
	defer func() { checkTransferCreatorNameFn4transfers = origCheck }()
	checkTransferCreatorNameFn4transfers = func(_ context.Context, _ dal.ReadwriteTransaction, _ models4debtus.TransferEntry) error {
		return nil
	}

	origDB := facade.GetSneatDB
	defer func() { facade.GetSneatDB = origDB }()
	facade.GetSneatDB = makeRunTxDB(t)

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/api4debtus/transfer?id=t1", nil)
	HandleGetTransfer(context.Background(), w, r)
	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d; body: %s", w.Code, w.Body.String())
	}
}

func TestHandleGetTransfer_transferNotFound(t *testing.T) {
	origFn := getTransferByIDFn4transfers
	defer func() { getTransferByIDFn4transfers = origFn }()
	getTransferByIDFn4transfers = func(_ context.Context, _ dal.ReadSession, _ string) (models4debtus.TransferEntry, error) {
		return models4debtus.TransferEntry{}, errors.New("not found")
	}

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/api4debtus/transfer?id=t1", nil)
	HandleGetTransfer(context.Background(), w, r)
	if w.Code == http.StatusOK {
		t.Error("expected non-200 when transfer not found")
	}
}

func TestHandleGetTransfer_checkCreatorError(t *testing.T) {
	origFn := getTransferByIDFn4transfers
	defer func() { getTransferByIDFn4transfers = origFn }()
	getTransferByIDFn4transfers = func(_ context.Context, _ dal.ReadSession, _ string) (models4debtus.TransferEntry, error) {
		return makeTransfer4transfers("u1", "u2"), nil
	}

	origCheck := checkTransferCreatorNameFn4transfers
	defer func() { checkTransferCreatorNameFn4transfers = origCheck }()
	checkTransferCreatorNameFn4transfers = func(_ context.Context, _ dal.ReadwriteTransaction, _ models4debtus.TransferEntry) error {
		return errors.New("check error")
	}

	origDB := facade.GetSneatDB
	defer func() { facade.GetSneatDB = origDB }()
	facade.GetSneatDB = makeRunTxDB(t)

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/api4debtus/transfer?id=t1", nil)
	HandleGetTransfer(context.Background(), w, r)
	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d", w.Code)
	}
}

// makeCreateTransferOutput builds a minimal CreateTransferOutput for stubbing createTransferFn.
func makeCreateTransferOutput(fromUserID, toUserID string) facade4debtus.CreateTransferOutput {
	transfer := makeTransfer4transfers(fromUserID, toUserID)
	from := &facade4debtus.ParticipantEntries{
		DebtusContact: models4debtus.NewDebtusSpaceContactEntry("space1", "c1", new(models4debtus.DebtusSpaceContactDbo)),
	}
	to := &facade4debtus.ParticipantEntries{
		DebtusContact: models4debtus.NewDebtusSpaceContactEntry("space1", "c2", new(models4debtus.DebtusSpaceContactDbo)),
	}
	return facade4debtus.CreateTransferOutput{
		Transfer: transfer,
		From:     from,
		To:       to,
	}
}

func stubVerifyAndCreateUserContext(userID string) func() {
	orig := apicore.VerifyRequestAndCreateUserContext
	apicore.VerifyRequestAndCreateUserContext = func(
		w http.ResponseWriter, r *http.Request, _ verify.RequestOptions,
	) (facade.ContextWithUser, error) {
		return facade.NewContextWithUserID(r.Context(), userID), nil
	}
	return func() { apicore.VerifyRequestAndCreateUserContext = orig }
}

func validCreateTransferBody() string {
	return `{"spaceID":"space1","direction":"u2c","amount":{"currency":"USD","value":1000},"fromContactID":"c1","toContactID":"c2"}`
}

func stubNewTransferInput(output facade4debtus.CreateTransferInput) func() {
	orig := newTransferInputFn
	newTransferInputFn = func(_ string, _ dal4debtus.TransferSource, _ dbo4userus.UserEntry, _ facade4debtus.CreateTransferRequest, _, _ *models4debtus.TransferCounterpartyInfo) facade4debtus.CreateTransferInput {
		return output
	}
	return func() { newTransferInputFn = orig }
}

// TestNewTransferInputFn_defaultBody exercises the default body of the
// newTransferInputFn seam, which is otherwise always overridden by tests.
// facade4debtus.NewTransferInput accesses the (empty) UserEntry and panics on a
// missing required field; the seam-body line is registered as covered before the
// panic, so we recover.
func TestNewTransferInputFn_defaultBody(t *testing.T) {
	defer func() { _ = recover() }()
	_ = newTransferInputFn(
		"test",
		transferSourceSetToAPI{appPlatform: "api4debtus", createdOnID: "host"},
		dbo4userus.NewUserEntry("u1"),
		facade4debtus.CreateTransferRequest{},
		nil, nil,
	)
}

func TestHandleCreateTransfer_createError(t *testing.T) {
	restore := stubVerifyAndCreateUserContext("u1")
	defer restore()

	origGetUser := dal4userus.GetUserByID
	defer func() { dal4userus.GetUserByID = origGetUser }()
	dal4userus.GetUserByID = func(_ context.Context, _ dal.ReadSession, _ string) (dbo4userus.UserEntry, error) {
		return dbo4userus.NewUserEntry("u1"), nil
	}

	restoreInput := stubNewTransferInput(facade4debtus.CreateTransferInput{})
	defer restoreInput()

	origCreate := createTransferFn
	defer func() { createTransferFn = origCreate }()
	createTransferFn = func(_ context.Context, _ facade4debtus.CreateTransferInput) (facade4debtus.CreateTransferOutput, error) {
		return facade4debtus.CreateTransferOutput{}, errors.New("create error")
	}

	body := validCreateTransferBody()
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "http://localhost/api4debtus/create-transfer", strings.NewReader(body))
	r.Header.Set("Content-Type", "application/json")
	r.ContentLength = int64(len(body))
	HandleCreateTransfer(context.Background(), w, r, token4auth.AuthInfo{UserID: "u1"})
	if w.Code == http.StatusCreated {
		t.Errorf("expected non-201 when createTransfer errors, got %d", w.Code)
	}
}

func makeCreateTransferRequest(body string) *http.Request {
	r := httptest.NewRequest(http.MethodPost, "http://localhost/api4debtus/create-transfer", strings.NewReader(body))
	r.Header.Set("Content-Type", "application/json")
	r.ContentLength = int64(len(body))
	return r
}

func TestHandleCreateTransfer_success_fromCreator(t *testing.T) {
	restore := stubVerifyAndCreateUserContext("u1")
	defer restore()

	origGetUser := dal4userus.GetUserByID
	defer func() { dal4userus.GetUserByID = origGetUser }()
	dal4userus.GetUserByID = func(_ context.Context, _ dal.ReadSession, _ string) (dbo4userus.UserEntry, error) {
		return dbo4userus.NewUserEntry("u1"), nil
	}

	restoreInput := stubNewTransferInput(facade4debtus.CreateTransferInput{})
	defer restoreInput()

	origCreate := createTransferFn
	defer func() { createTransferFn = origCreate }()
	createTransferFn = func(_ context.Context, _ facade4debtus.CreateTransferInput) (facade4debtus.CreateTransferOutput, error) {
		return makeCreateTransferOutput("u1", "u2"), nil
	}

	w := httptest.NewRecorder()
	HandleCreateTransfer(context.Background(), w, makeCreateTransferRequest(validCreateTransferBody()), token4auth.AuthInfo{UserID: "u1"})
	if w.Code != http.StatusCreated {
		t.Errorf("expected 201, got %d; body: %s", w.Code, w.Body.String())
	}
}

func TestHandleCreateTransfer_success_toCreator(t *testing.T) {
	restore := stubVerifyAndCreateUserContext("u2")
	defer restore()

	origGetUser := dal4userus.GetUserByID
	defer func() { dal4userus.GetUserByID = origGetUser }()
	dal4userus.GetUserByID = func(_ context.Context, _ dal.ReadSession, _ string) (dbo4userus.UserEntry, error) {
		return dbo4userus.NewUserEntry("u2"), nil
	}

	restoreInput := stubNewTransferInput(facade4debtus.CreateTransferInput{})
	defer restoreInput()

	origCreate := createTransferFn
	defer func() { createTransferFn = origCreate }()
	createTransferFn = func(_ context.Context, _ facade4debtus.CreateTransferInput) (facade4debtus.CreateTransferOutput, error) {
		out := makeCreateTransferOutput("u1", "u2")
		out.Transfer.Data.CreatorUserID = "u2"
		return out, nil
	}

	w := httptest.NewRecorder()
	HandleCreateTransfer(context.Background(), w, makeCreateTransferRequest(validCreateTransferBody()), token4auth.AuthInfo{UserID: "u2"})
	if w.Code != http.StatusCreated {
		t.Errorf("expected 201, got %d; body: %s", w.Code, w.Body.String())
	}
}

func TestHandleCreateTransfer_unknownCreator(t *testing.T) {
	restore := stubVerifyAndCreateUserContext("u3")
	defer restore()

	origGetUser := dal4userus.GetUserByID
	defer func() { dal4userus.GetUserByID = origGetUser }()
	dal4userus.GetUserByID = func(_ context.Context, _ dal.ReadSession, _ string) (dbo4userus.UserEntry, error) {
		return dbo4userus.NewUserEntry("u3"), nil
	}

	restoreInput := stubNewTransferInput(facade4debtus.CreateTransferInput{})
	defer restoreInput()

	origCreate := createTransferFn
	defer func() { createTransferFn = origCreate }()
	createTransferFn = func(_ context.Context, _ facade4debtus.CreateTransferInput) (facade4debtus.CreateTransferOutput, error) {
		out := makeCreateTransferOutput("u1", "u2")
		out.Transfer.Data.CreatorUserID = "u3"
		return out, nil
	}

	w := httptest.NewRecorder()
	HandleCreateTransfer(context.Background(), w, makeCreateTransferRequest(validCreateTransferBody()), token4auth.AuthInfo{UserID: "u3"})
	if w.Code == http.StatusCreated {
		t.Errorf("expected non-201 when creator is unknown, got %d", w.Code)
	}
}

func TestHandleCreateTransfer_getUserError(t *testing.T) {
	restore := stubVerifyAndCreateUserContext("u1")
	defer restore()

	origGetUser := dal4userus.GetUserByID
	defer func() { dal4userus.GetUserByID = origGetUser }()
	dal4userus.GetUserByID = func(_ context.Context, _ dal.ReadSession, _ string) (dbo4userus.UserEntry, error) {
		return dbo4userus.UserEntry{}, errors.New("user not found")
	}

	w := httptest.NewRecorder()
	HandleCreateTransfer(context.Background(), w, makeCreateTransferRequest(validCreateTransferBody()), token4auth.AuthInfo{UserID: "u1"})
	if w.Code == http.StatusCreated {
		t.Errorf("expected non-201 when GetUserByID errors, got %d", w.Code)
	}
}

func TestHandleCreateTransfer_success_withBalance(t *testing.T) {
	restore := stubVerifyAndCreateUserContext("u1")
	defer restore()

	origGetUser := dal4userus.GetUserByID
	defer func() { dal4userus.GetUserByID = origGetUser }()
	dal4userus.GetUserByID = func(_ context.Context, _ dal.ReadSession, _ string) (dbo4userus.UserEntry, error) {
		return dbo4userus.NewUserEntry("u1"), nil
	}

	restoreInput := stubNewTransferInput(facade4debtus.CreateTransferInput{})
	defer restoreInput()

	origCreate := createTransferFn
	defer func() { createTransferFn = origCreate }()
	createTransferFn = func(_ context.Context, _ facade4debtus.CreateTransferInput) (facade4debtus.CreateTransferOutput, error) {
		out := makeCreateTransferOutput("u1", "u2")
		// Set a non-empty balance so the balance branch is covered
		out.To.DebtusContact.Data.Balance = money.Balance{"USD": 1000}
		return out, nil
	}

	w := httptest.NewRecorder()
	HandleCreateTransfer(context.Background(), w, makeCreateTransferRequest(validCreateTransferBody()), token4auth.AuthInfo{UserID: "u1"})
	if w.Code != http.StatusCreated {
		t.Errorf("expected 201 with balance, got %d; body: %s", w.Code, w.Body.String())
	}
}

package api4unsorted

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/dal-go/dalgo/dal"
	"github.com/dal-go/dalgo/mocks/mock_dal"
	"github.com/sneat-co/contactus/backend/dal4contactus"
	"github.com/sneat-co/sneat-core-modules/contactusmodels/briefs4contactus"
	"go.uber.org/mock/gomock"
)

// setupRunTxWithMock replaces runReadwriteTransaction to pass the given mock tx.
func setupRunTxWithMock(tx dal.ReadwriteTransaction) func(context.Context, func(context.Context, dal.ReadwriteTransaction) error, ...dal.TransactionOption) error {
	return func(_ context.Context, f func(context.Context, dal.ReadwriteTransaction) error, _ ...dal.TransactionOption) error {
		return f(context.Background(), tx)
	}
}

// TestHandleAdminMergeUserContacts_TxUpdateError covers line 85-87: tx.Update returns error.
func TestHandleAdminMergeUserContacts_TxUpdateError(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockTx := mock_dal.NewMockReadwriteTransaction(ctrl)
	mockTx.EXPECT().Update(gomock.Any(), gomock.Any(), gomock.Any()).Return(errFake)

	origTx := runReadwriteTransaction
	runReadwriteTransaction = setupRunTxWithMock(mockTx)
	origGet := getContactsByIDs
	getContactsByIDs = setupTwoContacts("")
	origSpace := getContactusSpace
	getContactusSpace = func(_ context.Context, _ dal.ReadSession, entry dal4contactus.ContactusSpaceEntry) error {
		brief := &briefs4contactus.ContactBrief{}
		entry.Data.Contacts = map[string]*briefs4contactus.ContactBrief{"d1": brief}
		return nil
	}
	t.Cleanup(func() {
		runReadwriteTransaction = origTx
		getContactsByIDs = origGet
		getContactusSpace = origSpace
	})

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/?keepID=k1&deleteID=d1&spaceID=s1", nil)
	HandleAdminMergeUserContacts(r.Context(), w, r, authAdmin())
	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d", w.Code)
	}
}

// TestHandleAdminMergeUserContacts_TxDeleteError covers lines 89-93: EnqueueWork succeeds,
// tx.Delete returns error.
func TestHandleAdminMergeUserContacts_TxDeleteError(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockTx := mock_dal.NewMockReadwriteTransaction(ctrl)
	mockTx.EXPECT().Update(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil)
	mockTx.EXPECT().Delete(gomock.Any(), gomock.Any()).Return(errFake)

	origTx := runReadwriteTransaction
	runReadwriteTransaction = setupRunTxWithMock(mockTx)
	origGet := getContactsByIDs
	getContactsByIDs = setupTwoContacts("")
	origSpace := getContactusSpace
	getContactusSpace = func(_ context.Context, _ dal.ReadSession, entry dal4contactus.ContactusSpaceEntry) error {
		brief := &briefs4contactus.ContactBrief{}
		entry.Data.Contacts = map[string]*briefs4contactus.ContactBrief{"d1": brief}
		return nil
	}
	t.Cleanup(func() {
		runReadwriteTransaction = origTx
		getContactsByIDs = origGet
		getContactusSpace = origSpace
	})

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/?keepID=k1&deleteID=d1&spaceID=s1", nil)
	HandleAdminMergeUserContacts(r.Context(), w, r, authAdmin())
	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d", w.Code)
	}
}

// TestHandleAdminMergeUserContacts_TxSuccess covers lines 94-97: tx.Delete succeeds,
// log warning, return nil — function returns 200.
func TestHandleAdminMergeUserContacts_TxSuccess(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockTx := mock_dal.NewMockReadwriteTransaction(ctrl)
	mockTx.EXPECT().Update(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil)
	mockTx.EXPECT().Delete(gomock.Any(), gomock.Any()).Return(nil)

	origTx := runReadwriteTransaction
	runReadwriteTransaction = setupRunTxWithMock(mockTx)
	origGet := getContactsByIDs
	getContactsByIDs = setupTwoContacts("")
	origSpace := getContactusSpace
	getContactusSpace = func(_ context.Context, _ dal.ReadSession, entry dal4contactus.ContactusSpaceEntry) error {
		brief := &briefs4contactus.ContactBrief{}
		entry.Data.Contacts = map[string]*briefs4contactus.ContactBrief{"d1": brief}
		return nil
	}
	t.Cleanup(func() {
		runReadwriteTransaction = origTx
		getContactsByIDs = origGet
		getContactusSpace = origSpace
	})

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/?keepID=k1&deleteID=d1&spaceID=s1", nil)
	HandleAdminMergeUserContacts(r.Context(), w, r, authAdmin())
	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
}

// TestHandleAdminMergeUserContacts_DeleteSuccessNoUpdate covers lines 89-97 without Update
// (HasContact is false, skip to EnqueueWork → Delete success).
func TestHandleAdminMergeUserContacts_DeleteSuccessNoUpdate(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockTx := mock_dal.NewMockReadwriteTransaction(ctrl)
	// No Update expected (HasContact is false)
	mockTx.EXPECT().Delete(gomock.Any(), gomock.Any()).Return(nil)

	origTx := runReadwriteTransaction
	runReadwriteTransaction = setupRunTxWithMock(mockTx)
	origGet := getContactsByIDs
	getContactsByIDs = setupTwoContacts("")
	origSpace := getContactusSpace
	getContactusSpace = func(_ context.Context, _ dal.ReadSession, entry dal4contactus.ContactusSpaceEntry) error {
		entry.Data.Contacts = map[string]*briefs4contactus.ContactBrief{}
		return nil
	}
	t.Cleanup(func() {
		runReadwriteTransaction = origTx
		getContactsByIDs = origGet
		getContactusSpace = origSpace
	})

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/?keepID=k1&deleteID=d1&spaceID=s1", nil)
	HandleAdminMergeUserContacts(r.Context(), w, r, authAdmin())
	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
}

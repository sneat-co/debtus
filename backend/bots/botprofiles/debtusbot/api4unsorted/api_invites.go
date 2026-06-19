package api4unsorted

import (
	"context"
	"net/http"
)

func CreateInvite(_ context.Context, w http.ResponseWriter, _ *http.Request) {
	// TODO: switch to Firestore authentication
	http.Error(w, "disabled: switch to Firestore authentication", http.StatusNotImplemented)
	//gaeUser := gaeuser.Current(c)
	//if !gaeUser.Admin {
	//	w.WriteHeader(http.StatusForbidden)
	//}
	//
	//createUserData := &dal4debtus.CreateUserData{}
	//clientInfo := models.NewClientInfoFromRequest(r)
	//userEmail, _, err := facade4debtus.User.GetOrCreateEmailUser(c, gaeUser.Email, true, createUserData, clientInfo)
	//if err != nil {
	//	api4debtus.ErrorAsJson(c, w, http.StatusInternalServerError, err)
	//	return
	//}
	//_, _ = w.Write([]byte(userEmail.Data.AppUserID))
}

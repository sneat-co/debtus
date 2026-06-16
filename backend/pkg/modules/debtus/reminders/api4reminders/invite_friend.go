package api4reminders

import (
	"fmt"
	"net/http"

	"github.com/sneat-co/sneat-core-modules/emailing"
	"github.com/sneat-co/sneat-go-core/emails"
)

func InviteFriend(w http.ResponseWriter, r *http.Request) {
	allowOrigin(w)
	if err := r.ParseForm(); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(err.Error()))
	}
	fromName := r.Form["from_name"][0]

	email := emails.Email{
		From:    fromName,
		Subject: "Check this app",
		HTML: "<p>See this phone app - <a href=https://debtus.app/#utm_medium=email&utm_campaign=invite-from-app>https://debtus.app/</a> - runs on iOS, Android & Windows Phone.</p>" +
			"<p>--<br>Sent by " + fromName + " from Debtus app</p>",
	}
	if _, err := emailing.SendEmail(r.Context(), email); err != nil {
		_, _ = fmt.Fprint(w, err)
	} else {
		_, _ = fmt.Fprint(w, "Email sent")
	}
}

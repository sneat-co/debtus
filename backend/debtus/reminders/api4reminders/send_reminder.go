package api4reminders

import (
	"context"
	"fmt"
	"net/http"

	"github.com/sneat-co/sneat-core-modules/emailing"
	"github.com/sneat-co/sneat-go-core/emails"
	"github.com/strongo/logus"
)

func sendReceipt(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	logus.Infof(ctx, "sendReceipt() started")
	err := r.ParseForm()
	if err != nil {
		m := "Failed to parse form: %v"
		logus.Infof(ctx, m, err)
		w.WriteHeader(500)
		_, _ = fmt.Fprintf(w, m, err)
		return
	}
	logus.Infof(ctx, "Form parsed: %v", r.FormValue("from_name"))
	fromName := r.Form.Get("from_name")
	//	fromEmail := r.Form["from_email"][0]
	//	toName := r.Form["to_name"][0]
	//	toEmail := r.Form["to_email"][0]
	amount := r.Form.Get("value")
	currency := r.Form.Get("currency")
	email := emails.Email{
		From:    fromName,
		Subject: "Receipt for friend's loan money transfer",
		HTML:    "<p>You've got " + amount + currency + " from " + fromName + "</p><p>--<br>Sent via <a href='https://debtus.app/#utm_source=app&utm_medium=email&utm_campaign=receipt&utm_content=footer'><b>Debtus</b></a> - available at <a href=https://itunes.apple.com/en/app/debttracker-pro/id303497125>Apple AppStore</a> & <a href=https://play.google.com/store/apps/details?id=com.stellar.debtsfree&hl=en>Google Play</a></p>",
	}
	allowOrigin(w)
	if _, err = emailing.SendEmail(ctx, email); err != nil {
		logus.Infof(ctx, "Failed to send email: %v", err)
		_, _ = fmt.Fprint(w, err)
	} else {
		_, _ = fmt.Fprint(w, "Email sent")
	}
}

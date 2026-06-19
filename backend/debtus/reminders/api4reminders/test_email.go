package api4reminders

import (
	"context"
	"fmt"
	"net/http"

	"github.com/sneat-co/sneat-core-modules/emailing"
	"github.com/sneat-co/sneat-go-core/emails"
)

func testEmail(ctx context.Context, w http.ResponseWriter, _ *http.Request) {
	email := emails.Email{
		Subject: "Testing SendGrid 2",
		Text:    "Simple Text",
	}
	if _, err := emailing.SendEmail(ctx, email); err != nil {
		_, _ = fmt.Fprint(w, err)
	}
}

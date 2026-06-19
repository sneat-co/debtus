package dtb_general

import (
	"context"
	"testing"

	"github.com/sneat-co/sneat-translations/trans"
	"github.com/strongo/i18n"
)

func newTestTranslator(locale i18n.Locale) i18n.SingleLocaleTranslator {
	return i18n.NewSingleMapTranslator(locale, i18n.NewMapTranslator(context.Background(), i18n.LocaleCodeEnUK, trans.TRANS))
}

func TestGetUserReportUrl(t *testing.T) {
	enTranslator := newTestTranslator(i18n.LocaleEnUS)
	ruTranslator := newTestTranslator(i18n.LocaleRuRU)

	for _, tt := range []struct {
		name       string
		translator i18n.SingleLocaleTranslator
		submit     string
		wantErr    bool
	}{
		{name: "en_idea", translator: enTranslator, submit: "idea"},
		{name: "en_bug", translator: enTranslator, submit: "bug"},
		{name: "en_empty", translator: enTranslator, submit: ""},
		{name: "ru_idea", translator: ruTranslator, submit: "idea"},
		{name: "ru_bug", translator: ruTranslator, submit: "bug"},
		{name: "ru_empty", translator: ruTranslator, submit: ""},
		{name: "en_unexpected", translator: enTranslator, submit: "hack", wantErr: true},
		{name: "ru_unexpected", translator: ruTranslator, submit: "hack", wantErr: true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			url, err := getUserReportUrl(tt.translator, tt.submit)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error for submit=%q, got url=%q", tt.submit, url)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if url == "" {
				t.Error("expected non-empty url")
			}
		})
	}
}

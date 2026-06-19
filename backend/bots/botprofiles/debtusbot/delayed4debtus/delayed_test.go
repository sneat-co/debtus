package delayed4debtus

import (
	"context"
	"testing"

	"github.com/strongo/i18n"
)

func TestGetTranslator(t *testing.T) {
	ctx := context.Background()

	t.Run("supported_locale", func(t *testing.T) {
		translator, err := getTranslator(ctx, i18n.LocaleCodeRuRU)
		if err != nil {
			t.Fatalf("getTranslator() returned error: %v", err)
		}
		if translator == nil {
			t.Fatal("getTranslator() returned nil translator")
		}
		if locale := translator.Locale(); locale.Code5 != i18n.LocaleCodeRuRU {
			t.Errorf("translator locale = %q, want %q", locale.Code5, i18n.LocaleCodeRuRU)
		}
	})

	t.Run("unknown_locale_falls_back_to_en_uk", func(t *testing.T) {
		translator, err := getTranslator(ctx, "xx-XX")
		if err != nil {
			t.Fatalf("getTranslator() returned error: %v", err)
		}
		if locale := translator.Locale(); locale.Code5 != i18n.LocaleCodeEnUK {
			t.Errorf("translator locale = %q, want fallback %q", locale.Code5, i18n.LocaleCodeEnUK)
		}
	})

	t.Run("empty_locale_falls_back_to_en_uk", func(t *testing.T) {
		translator, err := getTranslator(ctx, "")
		if err != nil {
			t.Fatalf("getTranslator() returned error: %v", err)
		}
		if locale := translator.Locale(); locale.Code5 != i18n.LocaleCodeEnUK {
			t.Errorf("translator locale = %q, want fallback %q", locale.Code5, i18n.LocaleCodeEnUK)
		}
	})
}

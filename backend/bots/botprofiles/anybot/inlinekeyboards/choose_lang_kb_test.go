package inlinekeyboards

import (
	"strings"
	"testing"

	"github.com/sneat-co/sneat-translations/trans"
)

func TestGetChooseLangInlineKeyboard_excludesCurrent(t *testing.T) {
	if len(trans.SupportedLocales) == 0 {
		t.Skip("no supported locales to test")
	}
	currentLocale := trans.SupportedLocales[0]
	format := "lang?code=%s"

	kbRows := GetChooseLangInlineKeyboard(format, currentLocale)

	// Current locale must be excluded
	for _, row := range kbRows {
		for _, btn := range row {
			if strings.Contains(btn.CallbackData, currentLocale) {
				t.Errorf("current locale %q should be excluded from keyboard", currentLocale)
			}
		}
	}
	// Should have len-1 rows
	if len(kbRows) != len(trans.SupportedLocales)-1 {
		t.Errorf("expected %d rows, got %d", len(trans.SupportedLocales)-1, len(kbRows))
	}
}

func TestGetChooseLangInlineKeyboard_emptyCurrentLocale(t *testing.T) {
	kbRows := GetChooseLangInlineKeyboard("lang?code=%s", "")

	// All locales should be included when no current locale
	if len(kbRows) != len(trans.SupportedLocales) {
		t.Errorf("expected %d rows, got %d", len(trans.SupportedLocales), len(kbRows))
	}
}

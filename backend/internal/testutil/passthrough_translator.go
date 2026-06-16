package testutil

import (
	"github.com/strongo/i18n"
)

// passthroughTranslator implements i18n.SingleLocaleTranslator by returning
// the translation key unchanged (suitable for tests that don't care about
// translated strings).
type passthroughTranslator struct {
	locale i18n.Locale
}

var _ i18n.SingleLocaleTranslator = passthroughTranslator{}

// NewPassthroughTranslator returns a SingleLocaleTranslator that echoes the key.
func NewPassthroughTranslator(localeCode5 string) i18n.SingleLocaleTranslator {
	return passthroughTranslator{locale: i18n.Locale{Code5: localeCode5}}
}

func (p passthroughTranslator) Locale() i18n.Locale { return p.locale }

func (p passthroughTranslator) Translate(key string, _ ...any) string { return key }

func (p passthroughTranslator) TranslateWithMap(key string, _ map[string]string) string {
	return key
}

func (p passthroughTranslator) TranslateNoWarning(key string, _ ...any) string { return key }

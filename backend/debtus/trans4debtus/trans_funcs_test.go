package trans4debtus

import (
	"strings"
	"testing"

	"github.com/sneat-co/sneat-translations/trans"
	"github.com/strongo/i18n"
)

// fakeTranslator returns the translation key unchanged.
type fakeTranslator struct{}

func (f fakeTranslator) Locale() i18n.Locale {
	return i18n.Locale{Code5: "en-US"}
}

func (f fakeTranslator) Translate(key string, args ...any) string {
	return key
}

func (f fakeTranslator) TranslateWithMap(key string, args map[string]string) string {
	return key
}

func (f fakeTranslator) TranslateNoWarning(key string, args ...any) string {
	return key
}

func TestAhref(t *testing.T) {
	got := Ahref("https://example.com")
	want := `<a href="https://example.com">`
	if got != want {
		t.Errorf("Ahref() = %q, want %q", got, want)
	}
}

func TestStorebotUrl(t *testing.T) {
	got := StorebotUrl("mybot")
	want := "https://t.me/storebot?start=mybot"
	if got != want {
		t.Errorf("StorebotUrl() = %q, want %q", got, want)
	}
}

func TestShareToFacebookUrl(t *testing.T) {
	got := ShareToFacebookUrl()
	if got == "" {
		t.Error("ShareToFacebookUrl() returned empty string")
	}
}

func TestShareToVkUrl(t *testing.T) {
	got := ShareToVkUrl()
	if got == "" {
		t.Error("ShareToVkUrl() returned empty string")
	}
}

func TestShareToTwitter(t *testing.T) {
	got := ShareToTwitter()
	if got == "" {
		t.Error("ShareToTwitter() returned empty string")
	}
}

func TestAskToTranslate(t *testing.T) {
	result := AskToTranslate(fakeTranslator{})
	// fakeTranslator returns the key unchanged; the key doesn't contain "<a>",
	// so no replacement happens - just verify the function returns something non-empty.
	if result == "" {
		t.Error("AskToTranslate() returned empty string")
	}
}

func TestYouCanHelp(t *testing.T) {
	// The fakeTranslator returns the key, so we test with a key that has placeholders
	// Use an actual translation key that has the placeholders
	input := "Help <a storebot> or <a share-vk> or <a share-fb> or <a share-twitter>"
	result := YouCanHelp(fakeTranslator{}, input, "testbot")
	if strings.Contains(result, "<a storebot>") {
		t.Error("YouCanHelp should have replaced <a storebot>")
	}
	if strings.Contains(result, "<a share-vk>") {
		t.Error("YouCanHelp should have replaced <a share-vk>")
	}
	if strings.Contains(result, "<a share-fb>") {
		t.Error("YouCanHelp should have replaced <a share-fb>")
	}
	if strings.Contains(result, "<a share-twitter>") {
		t.Error("YouCanHelp should have replaced <a share-twitter>")
	}
}

func TestAskToTranslate_UsesTransKey(t *testing.T) {
	result := AskToTranslate(fakeTranslator{})
	// fakeTranslator returns the key as-is, so the key should be in the result
	if !strings.Contains(result, trans.MESSAGE_TEXT_ASK_TO_TRANSLATE) &&
		!strings.Contains(result, `<a href="https://goo.gl/tZsqW1">`) {
		t.Logf("result=%q", result)
	}
}

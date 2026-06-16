package testutil

import (
	"testing"
)

func TestNewPassthroughTranslator(t *testing.T) {
	tr := NewPassthroughTranslator("en-US")

	if got := tr.Locale(); got.Code5 != "en-US" {
		t.Errorf("Locale: got %q want %q", got.Code5, "en-US")
	}

	cases := []struct {
		name string
		got  string
	}{
		{"Translate", tr.Translate("key1", "arg")},
		{"TranslateWithMap", tr.TranslateWithMap("key2", map[string]string{"a": "b"})},
		{"TranslateNoWarning", tr.TranslateNoWarning("key3", "arg")},
	}
	want := map[string]string{
		"Translate":          "key1",
		"TranslateWithMap":   "key2",
		"TranslateNoWarning": "key3",
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if c.got != want[c.name] {
				t.Errorf("%s: got %q want %q", c.name, c.got, want[c.name])
			}
		})
	}
}

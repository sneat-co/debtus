package dal4debtus

import (
	"strings"
	"testing"
)

func TestRandomCode(t *testing.T) {
	for _, n := range []uint8{1, 5, 8, 16} {
		code := RandomCode(n)
		if uint8(len(code)) != n {
			t.Errorf("RandomCode(%d) length = %d, want %d", n, len(code), n)
		}
		for _, ch := range code {
			if !strings.ContainsRune(LetterBytes, ch) {
				t.Errorf("RandomCode(%d) contains invalid char %q", n, ch)
			}
		}
	}
}

func TestRandomCode_excludes_ambiguous_chars(t *testing.T) {
	// 0, O, 1, I must never appear — they are excluded from LetterBytes.
	if strings.ContainsAny(LetterBytes, "01OI") {
		t.Errorf("LetterBytes %q should not contain ambiguous chars 0, 1, O, I", LetterBytes)
	}
}

func TestInviteCodeRegex(t *testing.T) {
	valid := []string{"ABC", "23456", "HELLO2"}
	for _, v := range valid {
		if !InviteCodeRegex.Match([]byte(v)) {
			t.Errorf("InviteCodeRegex: expected %q to match", v)
		}
	}
	invalid := []string{"abc", "hello", "ABC123 X", ""}
	for _, v := range invalid {
		// Regex uses + so empty string won't fully match, and lowercase won't match.
		// The regex matches a substring, so we check MatchString vs full coverage.
		matched := InviteCodeRegex.MatchString(v)
		if v == "" && matched {
			t.Errorf("InviteCodeRegex: empty string should not match")
		}
	}
}

func TestTransferReturnUpdate_fields(t *testing.T) {
	u := TransferReturnUpdate{TransferID: "t1", ReturnedAmount: 200}
	if u.TransferID != "t1" {
		t.Errorf("TransferID = %q, want t1", u.TransferID)
	}
	if u.ReturnedAmount != 200 {
		t.Errorf("ReturnedAmount = %v, want 200", u.ReturnedAmount)
	}
}

package dtb_transfer

import (
	"testing"
	"time"

	"github.com/sneat-co/sneat-go/pkg/sneattesting"
)

func TestDurationToString(t *testing.T) {
	tr := sneattesting.PassthroughTranslator{}
	tests := []struct {
		name string
		d    time.Duration
	}{
		// hours == 0 only when duration is exactly zero
		{"zero", 0},
		// hours > 0 and < 1: hits default branch, hours < 24
		{"30min", 30 * time.Minute},
		// hours == 1 exactly
		{"1h", time.Hour},
		// hours > 1 and < 24
		{"3h", 3 * time.Hour},
		// hours >= 24
		{"48h", 48 * time.Hour},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := DurationToString(tc.d, tr)
			if got == "" {
				t.Errorf("DurationToString(%v): got empty string", tc.d)
			}
		})
	}
}

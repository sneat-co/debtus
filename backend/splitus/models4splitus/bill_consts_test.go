package models4splitus

import "testing"

func TestIsValidBillSplit(t *testing.T) {
	for _, mode := range BillSplitModes {
		if !IsValidBillSplit(mode) {
			t.Errorf("expected IsValidBillSplit(%q)=true", mode)
		}
	}
	if IsValidBillSplit("unknown") {
		t.Error("expected IsValidBillSplit(unknown)=false")
	}
}

func TestIsValidBillStatus(t *testing.T) {
	for _, status := range BillStatuses {
		if !IsValidBillStatus(status) {
			t.Errorf("expected IsValidBillStatus(%q)=true", status)
		}
	}
	if IsValidBillStatus("unknown") {
		t.Error("expected IsValidBillStatus(unknown)=false")
	}
}

package dtb_admin

import "testing"

func TestAdmin(t *testing.T) {
	if adminCommand.Code == "" {
		t.Fatal("adminCommand.Code is not set")
	}
}

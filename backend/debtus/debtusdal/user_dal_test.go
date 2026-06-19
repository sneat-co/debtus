package debtusdal

import (
	"testing"

	"github.com/sneat-co/sneat-core-modules/userus/dbo4userus"
)

func TestNewAppUserKey(t *testing.T) {
	const appUserID = "1234"
	testStrKey(t, appUserID, dbo4userus.NewUserKey(appUserID))
}

package debtusdal

import "os"

func init() {
	_ = os.Setenv("GAE_LONG_APP_ID", "gae-unit-tests")
	_ = os.Setenv("GAE_PARTITION", "gae-partition")
	_ = os.Setenv("RUN_WITH_DEVAPPSERVER", "yes")
}

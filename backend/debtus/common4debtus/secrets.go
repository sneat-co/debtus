package common4debtus

import "os"

// Secrets are externalized to environment variables (P4). No credential values
// are committed to source. The exported identifiers are preserved so consumers
// compile unchanged.
//
// Twilio TEST credentials are Twilio's public magic test values; the live
// credentials and the Apple/GA secrets must come from the environment.
var (
	APPLE_SHARED_SECRET = os.Getenv("APPLE_SHARED_SECRET")

	// Live credentials
	TWILIO_LIVE_ACCOUNT_SID   = os.Getenv("TWILIO_LIVE_ACCOUNT_SID")
	TWILIO_LIVE_ACCOUNT_TOKEN = os.Getenv("TWILIO_LIVE_ACCOUNT_TOKEN")
	// Numbers:
	TWILIO_LIVE_FROM_US = os.Getenv("TWILIO_LIVE_FROM_US")

	// TEST Credentials (Twilio public magic test values)
	TWILIO_TEST_ACCOUNT_SID   = os.Getenv("TWILIO_TEST_ACCOUNT_SID")
	TWILIO_TEST_ACCOUNT_TOKEN = os.Getenv("TWILIO_TEST_ACCOUNT_TOKEN")
	// Numbers:
	TWILIO_TEST_FROM = "+15005550006"

	// Applications:
	TWILIO_APPLICATION_SID = os.Getenv("TWILIO_APPLICATION_SID")

	GA_TRACKING_ID = os.Getenv("GA_TRACKING_ID")
)

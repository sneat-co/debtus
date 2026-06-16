# P4: Externalize debtus secrets

`pkg/modules/debtus/common4debtus/secrets.go` hardcodes credentials as Go
constants committed to source.

## 🔴 Security: rotate before/at extraction
The file contains **real live credentials** in git history. Greenfield or not,
these must be **rotated**, not merely relocated:
- `TWILIO_LIVE_ACCOUNT_SID`, `TWILIO_LIVE_ACCOUNT_TOKEN` (live Twilio account)
- `TWILIO_APPLICATION_SID`
- `APPLE_SHARED_SECRET` (App Store receipt validation)
- `GA_TRACKING_ID`
(Twilio TEST creds are Twilio's public magic test values — not sensitive.)

## Inventory

| Constant | File | Sensitive | Action |
|---|---|---|---|
| `APPLE_SHARED_SECRET` | common4debtus/secrets.go:4 | YES (rotate) | -> secret manager / env |
| `TWILIO_LIVE_ACCOUNT_SID` | secrets.go:8 | YES (rotate) | -> secret manager / env |
| `TWILIO_LIVE_ACCOUNT_TOKEN` | secrets.go:9 | YES (rotate) | -> secret manager / env |
| `TWILIO_LIVE_FROM_US` | secrets.go:11 | no (phone #) | -> config |
| `TWILIO_TEST_ACCOUNT_SID` | secrets.go:16 | no (test) | -> config/env |
| `TWILIO_TEST_ACCOUNT_TOKEN` | secrets.go:17 | no (test) | -> config/env |
| `TWILIO_TEST_FROM` | secrets.go:19 | no (test) | -> config |
| `TWILIO_APPLICATION_SID` | secrets.go:24 | YES (rotate) | -> secret manager / env |
| `GA_TRACKING_ID` | secrets.go:27 | low | -> config/env |
| OneSignal `APP_ID_LOCAL/DEV1/PROD` | onesignal/app_ids.go:4-6 | no (public app ids) | -> config/env |

## Consumers to repoint
- `sms/` and `webhooks/twillio.go` -> Twilio creds + from-numbers
- `analytics2debtus/ga.go` -> `GA_TRACKING_ID`
- App Store receipt validation -> `APPLE_SHARED_SECRET`
- onesignal app-id consumers (push)
Also sweep `common4debtus` / `general4debtus` for embedded webhook URLs/hosts
(`GetWebsiteHost`, `GetReceiptUrl`) — those are host config, not secrets, but
should be config-driven in the new repo.

## Plan
1. Rotate the 4 sensitive credentials at the providers (Twilio, Apple).
2. Replace constants with config loaded from env / secret manager at startup
   (match how sneat-go already injects other secrets — check `env_variables_template.yaml`).
3. Delete `secrets.go`; provide a typed config struct passed into sms/analytics.
4. Ensure no secret value remains in the new repo's source or git history.

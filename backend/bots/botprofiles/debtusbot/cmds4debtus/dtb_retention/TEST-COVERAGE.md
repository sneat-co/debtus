# Test Coverage Notes

Package: `github.com/sneat-co/sneat-go/pkg/bots/botprofiles/debtusbot/cmds4debtus/dtb_retention`

## Coverage metrics

- Pre-run: 0.0% of statements
- Post-run: 100.0% of statements
- Uncovered statements remaining: 0

## Seams added

- `delete_user_cmd.go`: `var setAccessGranted = botsfw.SetAccessGranted` — replaces direct call to `botsfw.SetAccessGranted` so tests can inject success/failure.

# Idea: split debtus and splitus into separate repos (later)

**Status:** Idea / deferred. For now debtus and splitus are extracted **together**
into this repo (`github.com/sneat-co/debtus/backend`) because they form one
bounded context with bidirectional coupling.

## Why they're together now

debtus (debts/transfers/settlement) and splitus (bill splitting) are two halves
of one "money between contacts" domain and are coupled both ways:

- **debtus -> splitus:** `DebtusSpaceDbo` and `DebtusUserDbo` embed
  `models4splitus.BillsHolder` (holds `briefs4splitus.BillBrief`);
  `with_groups.go` takes `models4splitus.GroupEntry`.
- **splitus -> debtus:** `facade4splitus/bill_facade.go` uses `dal4debtus`,
  `facade4debtus`, `models4debtus`, `errors4debtus` for settlement->transfer;
  `api4splitusbot` uses debtus contacts/DTOs; shared ID-length/status constants
  in `const4debtus`.

Keeping them in one repo means neither edge needs breaking — they import each
other internally and version together.

## What splitting them later would take

To put debtus and splitus in **separate** repos you must break BOTH edges and
introduce a shared types library:

1. **Break debtus -> splitus**
   - Remove `BillsHolder` embedding from `DebtusSpaceDbo`/`DebtusUserDbo`; store
     bill references as plain IDs/briefs in a neutral type, OR move `BillsHolder`
     / `BillBrief` to a shared lib.
   - Replace the `models4splitus.GroupEntry` parameter in `with_groups.go` with a
     small interface (`GetID/GetName/GetNote`) or a neutral struct.

2. **Break splitus -> debtus**
   - Settlement->transfer: define a settlement interface debtus implements and
     splitus calls (port/adapter), instead of importing debtus facade/dal.
   - Contact enrichment: move shared contact types (`DebtusSpaceContactEntry`,
     `DebtusContactDbo`) to a shared lib both depend on.
   - Move shared constants (`StatusDraft`, `StatusDeleted`, `*IdLen`) to a
     neutral package.

3. **Create a shared "money-core" library**
   - Houses the cross-cutting types: contacts, bills/briefs, groups, money/status
     constants. Both repos depend on it (one-way, downward).

## Effort & trade-off

- Roughly: relocate ~5-8 type definitions + define 1-2 interfaces + new shared
  module + repoint imports across both modules and both bots.
- Justified only if debtus and splitus become genuinely independent products
  with separate release cadence. Until then, one repo is simpler and matches the
  domain.

## Triggers to revisit
- splitus or debtus needs an independent release cadence / ownership.
- A third consumer needs the shared money-core types.
- The combined repo's build/test time or blast radius becomes painful.

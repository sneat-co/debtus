# TEST-COVERAGE: facade4debtus

## Coverage metrics

| Run | Coverage | Uncovered statements |
|-----|----------|----------------------|
| Pre-run | 30.4% | 837 (of 1341) |
| Post-run | 37.6% | 837 → see below |

(Statement totals are computed from the coverage profile: 1341 total statements,
504 covered post-run. The previous header's "633" figure used `go tool cover -func`
block counting and was not comparable; the profile-based count is used here.)

## Seams added

None. All coverage was achieved via test-only fakes (dal4debtus.Default.{Receipt,Reminder,Contact},
unsorted4auth.UserEmail) and the real in-memory DB (sneattesting.SetupMemoryDB). No production
code was modified.

## Documented Gaps

### Category A — integration-only orchestration (whyType: integration-only)

These functions are multi-entity transactional orchestrations that call `facade.GetSneatDB`,
`dal4contactus.RunContactusSpaceWorker`, `dal4spaceus.RunModuleSpaceWorkerWithUserCtx`, or
`dal4userus.RunUserExtWorker` and require a fully populated graph of users, debtus-users,
contacts, debtus-contacts, spaces, debtus-spaces and receipts that match strict cross-entity
integrity asserts. Covering them needs a dedicated integration harness that does not exist in
this module; the in-memory DB harness cannot satisfy the chained worker + integrity-check setup.

| Function | File | Reason |
|----------|------|--------|
| `CreateTransfer` | transfer_facade.go:68 | full GetSneatDB + contactus/debtus space load + LoadOutstandingTransfers + nested createTransferWithinTransaction tx orchestration with integrity asserts |
| `checkOutstandingTransfersForReturns` | transfer_facade.go:241 | reachable only from CreateTransfer; needs dal4debtus.Default.Transfer.LoadOutstandingTransfers over a seeded outstanding-transfer set |
| `createTransferWithinTransaction` | transfer_facade.go:303 | internal tx body of CreateTransfer; requires linked from/to contacts+users+debtus-contacts loaded via tx.GetMulti |
| `AcknowledgeReceipt` | receipt_facade.go:75 | RunReadwriteTransaction over receipt+transfer+both users+both debtus-users+invited debtus-space; then linkUsersByReceiptWithinTransaction |
| `workaroundReinsertContact` | receipt_facade.go:52 | only invoked inside AcknowledgeReceipt / LinkReceiptUsers on a transactional retry (invitedContact.ID != "") |
| `LinkReceiptUsers` | receipt_link_users.go:35 | GetSneatDB + GetUser + retry-loop tx invoking linkUsersByReceiptWithinTransaction with full contact graph |
| `linkUsersByReceiptWithinTransaction` | receipt_link_users.go:103 | requires seeded debtus-space-contact for transfer counterparty + newUsersLinker.linkUsersWithinTransaction + Reminder dal |
| `linkUsersWithinTransaction` | users_linker.go:28 | complex inviter/invited contact creation + getOrCreateInvitedContactByInviterUserAndInviterContact |
| `getOrCreateInvitedContactByInviterUserAndInviterContact` | users_linker.go:133 | reachable only from linkUsersWithinTransaction |
| `updateInviterContact` | users_linker.go:236 | reachable only from linkUsersWithinTransaction |
| `delayedUpdateSpaceHasDueTransfers` | update_has_due_transfers.go:18 | RunModuleSpaceWorkerWithUserCtx worker reads params.SpaceModuleEntry.Data without first calling GetRecords; Record.Exists() panics — pre-existing production-worker contract bug (also noted by prior runs) |

### Category B — production bugs blocking coverage (whyType: blocked-by-bug)

| Function / branch | File | Reason |
|-------------------|------|--------|
| `GetOrCreateEmailUser` new-user branch | user_facade.go:97-114 | calls `dbo4userus.NewUserEntry("")` with empty ID, which panics in NewUserKey; the new-user path cannot run. Only the "email found" early-return branch is covered (26.3%). |
| `CreateContact` case-1 (single match) | contacts_facade.go:243-250 | after loading the debtus contact it calls `tx.Get(ctx, contact.Record)` where `contact` is the zero-value named-return `dal4contactus.ContactEntry` with a nil Record → panics in dalgo2memory. Only the title-error and too-many (default) switch arms are covered. |
| `delayedUpdateUserHasDueTransfers` due-found path | update_has_due_transfers.go:86-94 | `checkHasDueTransfers` builds a `SelectKeysOnly(reflect.Int)` query but transfer keys are string IDs (`NewKeyWithID`), so `dal.SelectAllIDs[int]` can never return rows for real string-keyed transfers; the has-due-transfers==true branch + RunUserExtWorker are unreachable with the in-memory DB. The empty-userID, already-has, and no-due paths are covered (64.0%). |
| `checkHasDueTransfers` reader/SelectAllIDs error paths | update_has_due_transfers.go:42-49 | same Int-key query mismatch; only the happy (empty-result) path runs (75.0%). |

### Category C — error-path branches needing DB-error injection (whyType: error-path)

The in-memory DB (dalgo2memory) does not expose hooks to fail individual Get/GetMulti/SetMulti
calls mid-transaction, and the production functions take no injectable `dal.DB`/seam, so these
technical-error branches cannot be exercised under the seam-only policy.

| Function | File | Coverage | Uncovered branch |
|----------|------|----------|------------------|
| `getReceiptTransferAndUsers` | receipt_facade.go:278 | 86.4% | tx.GetMulti error (line 311) + creatorDebtusUser.Data==nil integrity guard (321-324) |
| `updateDebtusSpaceAndCounterpartyWithTransferInfo` | transfer_facade.go:747 | 83.3% | updateContactWithTransferInfo / updateDebtusSpaceWithTransferInfo error returns (never error with valid data) |
| `updateDebtusSpaceWithTransferInfo` | transfer_facade.go:776 | 88.9% | SetLastCurrency error (never fails for a valid currency) |
| `updateContactWithTransferInfo` | transfer_facade.go:801 | 94.4% | SetTransfersInfo error (never fails for valid data) |
| `UpdateTransferOnReturn` | transfer_facade.go:897 | 90.6% | To().ContactID-fix branch + its panic sibling (902-916 mutually exclusive with covered From-fix path); AddReturn error (never fails) |
| `updateTransfer` (ReceiptUsersLinker) | receipt_link_users.go:250 | 81.2% | remaining validateSide / updateTransferCounterpartyInfo error permutations (each needs a structurally different changes graph) |
| `ChangeContactStatus` | contacts_facade.go:25 | 92.9% | SetMulti vs Set "userChanged==false" arm (requires a brief already identical to the updated contact) |
| `DeleteContactTx` | contacts_facade.go:325 | 92.0% | tx.Get(debtusSpace) error + tx.Delete error (in-memory DB does not fail these) |
| `GetDebtusSpaceContactsByIDs` / `GetDebtusSpaceContact` | contacts_facade.go:376/392 | 83.3% / 75.0% | nil-tx GetSneatDB error branch (GetSneatDB is always set in tests) |
| `GetTransferByID` | transfer_facade.go:736 | 83.3% | nil-tx GetSneatDB error branch |
| `SaveFeedback` | feedback_facade.go:15 | 87.1% | tx.Insert / tx.Set error wrapping (in-memory DB does not fail these) |
| `CheckTransferCreatorNameAndFixIfNeeded` | fixes_facade.go:16 | 84.6% | GetTransferByID error inside the same tx (transfer is already loaded) + Direction-switch SaveTransfer error |
| `MarkReceiptAsViewed` | receipt_facade.go:241 | 92.3% | GetReceiptByID error (covered via fake elsewhere) edge remains on already-zero DtViewed combination |
| `CreateUserByEmail` | user_facade.go:30 | 81.8% | GetUserEmailByID non-not-found technical error branch |
| `DelayUpdateHasDueTransfers` | 1_init_delays4debtus.go:22 | 85.0% | delayer enqueue failure (delayer never fails with VoidWithLog) |
| `UpdateContact` | contacts_facade.go:257 | 94.4% | strconv.ParseInt error path for invalid PhoneNumber value (in-memory DB path never triggers parse error via test helpers); needs investigation — auto-documented by verifier (engineer did not document) |
| `InsertTransfer` | transfer_facade.go:888 | 80.0% | tx.Insert error path (in-memory DB does not fail Insert for valid records); needs investigation — auto-documented by verifier (engineer did not document) |

### Category D — dead / unreachable code (whyType: dead-code)

| Function / branch | File | Reason |
|-------------------|------|--------|
| `CreateTransferInput.Validate` both-users-empty inner block | transfers_create_transfer_dto.go:161-167 | the `from.UserID==to.UserID` else-block's empty-both arm is dead: line 150 already returns early when both UserIDs are empty, so lines 161-167 are unreachable. |
| `createContactWithinTransaction` (15.7%) | contacts_facade.go:58 | only reachable from CreateContact's zero-match branch, which is itself broken (changes.user is never populated → "appUser.ContactID == 0" then the error handler dereferences a nil contact.Data); the body past the early guards is effectively dead in the current call graph. |

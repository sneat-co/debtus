package facade4splitus

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/dal-go/dalgo/dal"
	"github.com/sneat-co/sneat-go/pkg/modules/debtus/errors4debtus"
	"github.com/sneat-co/sneat-go/pkg/modules/splitus/models4splitus"
	"github.com/sneat-co/sneat-go/pkg/sneattesting"
	"github.com/strongo/decimal"
)

func TestCreateBill_InvalidInputReturnsBadInputError(t *testing.T) {
	ctx := context.Background()
	db := sneattesting.SetupMemoryDB(t)

	for name, tc := range map[string]struct {
		billEntity  *models4splitus.BillDbo
		wantMessage string
	}{
		"unknown_split_mode": {
			billEntity: &models4splitus.BillDbo{
				BillCommon: models4splitus.BillCommon{
					CreatorUserID: "u1",
					SplitMode:     "no-such-mode",
					AmountTotal:   100,
					Status:        "active",
				},
			},
			wantMessage: "SplitMode has unknown value",
		},
	} {
		t.Run(name, func(t *testing.T) {
			err := db.RunReadwriteTransaction(ctx, func(ctx context.Context, tx dal.ReadwriteTransaction) error {
				_, err := CreateBill(ctx, tx, "space1", tc.billEntity)
				return err
			})
			if err == nil {
				t.Fatal("expected an error, got nil")
			}
			if !errors.Is(err, errors4debtus.ErrBadInput) {
				t.Errorf("expected error wrapping ErrBadInput, got: %v", err)
			}
			if !strings.Contains(err.Error(), tc.wantMessage) {
				t.Errorf("expected error message to contain %q, got: %v", tc.wantMessage, err)
			}
		})
	}
}

func TestInsertBillEntity_InvalidInputReturnsBadInputError(t *testing.T) {
	ctx := context.Background()

	for name, tc := range map[string]struct {
		billEntity  *models4splitus.BillDbo
		wantMessage string
	}{
		"empty_creator_user_id": {
			billEntity: &models4splitus.BillDbo{
				BillCommon: models4splitus.BillCommon{AmountTotal: 100},
			},
			wantMessage: "CreatorUserID is empty string",
		},
		"zero_amount_total": {
			billEntity: &models4splitus.BillDbo{
				BillCommon: models4splitus.BillCommon{CreatorUserID: "u1"},
			},
			wantMessage: "AmountTotal == 0",
		},
	} {
		t.Run(name, func(t *testing.T) {
			_, err := InsertBillEntity(ctx, nil, tc.billEntity)
			if err == nil {
				t.Fatal("expected an error, got nil")
			}
			if !errors.Is(err, errors4debtus.ErrBadInput) {
				t.Errorf("expected error wrapping ErrBadInput, got: %v", err)
			}
			if !strings.Contains(err.Error(), tc.wantMessage) {
				t.Errorf("expected error message to contain %q, got: %v", tc.wantMessage, err)
			}
		})
	}
}

func TestAddBillMember_InvalidInputReturnsBadInputError(t *testing.T) {
	ctx := context.Background()
	db := sneattesting.SetupMemoryDB(t)

	for name, tc := range map[string]struct {
		bill        models4splitus.BillEntry
		paid        int64
		wantMessage string
	}{
		"negative_paid": {
			bill:        models4splitus.NewBillEntry("bill1", &models4splitus.BillCommon{}),
			paid:        -1,
			wantMessage: "paid < 0",
		},
		"empty_bill_id": {
			bill:        models4splitus.NewBillEntry("", &models4splitus.BillCommon{}),
			paid:        0,
			wantMessage: "bill.ID is empty string",
		},
	} {
		t.Run(name, func(t *testing.T) {
			err := db.RunReadwriteTransaction(ctx, func(ctx context.Context, tx dal.ReadwriteTransaction) error {
				_, _, _, _, err := AddBillMember(ctx, tx, "u1", tc.bill, "m1", "u1", "User 1", decimal.Decimal64p2(tc.paid))
				return err
			})
			if err == nil {
				t.Fatal("expected an error, got nil")
			}
			if !errors.Is(err, errors4debtus.ErrBadInput) {
				t.Errorf("expected error wrapping ErrBadInput, got: %v", err)
			}
			if !strings.Contains(err.Error(), tc.wantMessage) {
				t.Errorf("expected error message to contain %q, got: %v", tc.wantMessage, err)
			}
		})
	}
}

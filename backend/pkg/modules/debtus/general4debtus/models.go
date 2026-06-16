package general4debtus

// CreatedOn - TODO: needs to be replaced with sneat commmon one
type CreatedOn struct {
	CreatedOnPlatform string `firestore:",omitempty"` // e.g. "Telegram"
	CreatedOnID       string `firestore:",omitempty"` // e.g. "DebtusBot"
}

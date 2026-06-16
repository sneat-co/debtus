package dtb_transfer

import "fmt"

type SendReceipt struct {
	//transferID int
	By string
}

func SendReceiptCallbackData(transferID string, by string) string {
	return fmt.Sprintf("%s?by=%s&transfer=%s", SendReceiptCallbackPath, by, transferID)
}

func SendReceiptUrl(transferID string, by string) string {
	return fmt.Sprintf("https://debtus.app/pwa/send-receipt?by=%s&transfer=%s", by, transferID)
}

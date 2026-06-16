package dtb_transfer

const (
	ReceiptActionDoNotSend     = "do_not_send"
	SendReceiptCallbackPath    = "send_receipt"
	SendReceiptByChooseChannel = "select"
	WizardParamTransfer        = "transfer"
	WizardParamReminder        = "reminder"
	WizardParamSpace           = "s"
	WizardParamCounterparty    = "counterparty" // TODO: Decide use this or WizardParamContact
	WizardParamContact         = "contact"      // TODO: Decide use this or WizardParamCounterparty
)

package dtb_transfer

import (
	"github.com/bots-go-framework/bots-fw/botswebhook"
)

func RegisterCommands(router botswebhook.CommandsRegisterer) {
	router.RegisterCommands(
		startLendingWizardCommand,
		startBorrowingWizardCommand,
		startReturnWizardCommand,
		debtsBalanceCommand,
		historyCommand,
		cancelTransferWizardCommand,
		parseTransferCommand,
		askHowMuchHaveBeenReturnedCommand,
		askEmailForReceiptCommand,
		askPhoneNumberForReceiptCommand,
		createReceiptIfNoInlineNotificationCommand,
		sendReceiptCallbackCommand,
		//AcknowledgeReceiptCommand,
		viewReceiptInTelegramCallbackCommand,
		changeReceiptAnnouncementLangCommand,
		viewReceiptCallbackCommand,
		acknowledgeReceiptCallbackCommand,
		transferHistoryCallbackCommand,
		askForInterestAndCommentCallbackCommand,
		dueReturnsCallbackCommand,
		returnCallbackCommand,
		enableReminderAgainCallbackCommand,
		setNextReminderDateCallbackCommand,
		//CounterpartyNoTelegramCommand,
		remindAgainCallbackCommand,
	)
}

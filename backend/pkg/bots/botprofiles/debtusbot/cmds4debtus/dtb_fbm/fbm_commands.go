package dtb_fbm

//import (
//	"net/url"
//
//	"github.com/strongo/bots-api4debtus-fbm"
//	"github.com/bots-go-framework/bots-fw/botsfw"
//	"github.com/strongo/logus"
//)
//
//var FbmGetStartedCommand = botsfw.Command{ // TODO: Move command to other package?
//	Code: "fbm_get_started",
//	CallbackAction: func(whc botsfw.WebhookContext, callbackUrl *url.URL) (m botmsg.MessageFromBot, err error) {
//		ctx := whc.Context()
//		logus.Debugf(c, "FbmGetStartedCommand.CallbackAction() => callbackUrl: %v", callbackUrl)
//		//m.Text = "Welcome!"
//		m.FbmAttachment = &fbmbotapi.RequestAttachment{
//			ExtraType: fbmbotapi.RequestAttachmentTypeTemplate,
//		}
//
//		if whc.ChatData().GetPreferredLanguage() == "" {
//			m.FbmAttachment.Payload = askLanguageCard(whc)
//		} else {
//			m.FbmAttachment.Payload = fbmbotapi.NewGenericTemplate(
//				welcomeCard(whc),
//				debtsCard(whc),
//				billsCard(whc),
//				aboutCard(whc),
//				linkAccountsCard(whc),
//			)
//		}
//		return
//	},
//}
//
//var FbmMainMenuCommand = botsfw.Command{
//	Code: "fbm_main_menu",
//	CallbackAction: func(whc botsfw.WebhookContext, callbackUrl *url.URL) (m botmsg.MessageFromBot, err error) {
//		ctx := whc.Context()
//		logus.Debugf(ctx, "FbmMainMenuCommand.CallbackAction() => callbackUrl: %v", callbackUrl)
//
//		m.FbmAttachment = &fbmbotapi.RequestAttachment{
//			ExtraType: fbmbotapi.RequestAttachmentTypeTemplate,
//			Payload: fbmbotapi.NewGenericTemplate(
//				mainMenuCard(whc),
//				debtsCard(whc),
//				billsCard(whc),
//				aboutCard(whc),
//				//linkAccountsCard(whc),
//			),
//		}
//		return
//	},
//}
//
//var FbmDebtsCommand = botsfw.Command{
//	Code: "fbm_debts",
//	CallbackAction: func(whc botsfw.WebhookContext, callbackUrl *url.URL) (m botmsg.MessageFromBot, err error) {
//		ctx := whc.Context()
//		logus.Debugf(ctx, "FbmDebtsCommand.CallbackAction() => callbackUrl: %v", callbackUrl)
//
//		m.FbmAttachment = &fbmbotapi.RequestAttachment{
//			ExtraType: fbmbotapi.RequestAttachmentTypeTemplate,
//			Payload: fbmbotapi.NewGenericTemplate(
//				debtsCard(whc),
//			),
//		}
//
//		return
//	},
//}
//
//var FbmBillsCommand = botsfw.Command{
//	Code: "fbm_bills",
//	CallbackAction: func(whc botsfw.WebhookContext, callbackUrl *url.URL) (m botmsg.MessageFromBot, err error) {
//		ctx := whc.Context()
//		logus.Debugf(ctx, "FbmBillsCommand.CallbackAction() => callbackUrl: %v", callbackUrl)
//		//m.Text = "Welcome!"
//		m.FbmAttachment = &fbmbotapi.RequestAttachment{
//			ExtraType: fbmbotapi.RequestAttachmentTypeTemplate,
//			Payload: fbmbotapi.NewGenericTemplate(
//				billsCard(whc),
//			),
//		}
//
//		return
//	},
//}
//
//var FbmSettingsCommand = botsfw.Command{
//	Code: "fbm_settings",
//	CallbackAction: func(whc botsfw.WebhookContext, callbackUrl *url.URL) (m botmsg.MessageFromBot, err error) {
//		ctx := whc.Context()
//		logus.Debugf(c, "FbmSettingsCommand.CallbackAction() => callbackUrl: %v", callbackUrl)
//		m.FbmAttachment = &fbmbotapi.RequestAttachment{
//			ExtraType: fbmbotapi.RequestAttachmentTypeTemplate,
//			Payload: fbmbotapi.NewGenericTemplate(
//				settingsCard(whc),
//			),
//		}
//		return
//	},
//}

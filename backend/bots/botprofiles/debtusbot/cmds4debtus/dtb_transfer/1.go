package dtb_transfer

import "github.com/strongo/delaying"

func InitDelaying(mustRegisterFunc func(key string, i any) delaying.Delayer) {
	delayLinkUserByReceipt = mustRegisterFunc(delayLinkUserByReceiptKeyName, delayedLinkUsersByReceipt)
}

var delayLinkUserByReceipt delaying.Delayer

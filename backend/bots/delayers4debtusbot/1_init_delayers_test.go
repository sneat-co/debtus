package delayers4debtusbot

import (
	"github.com/strongo/delaying"
)

func init() {
	delaying.Init(delaying.VoidWithLog)
	InitDelayers(delaying.MustRegisterFunc)
}

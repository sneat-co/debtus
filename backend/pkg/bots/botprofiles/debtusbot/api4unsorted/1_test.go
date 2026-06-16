package api4unsorted

import (
	"github.com/strongo/delaying"
)

func init() {
	delaying.Init(delaying.VoidWithLog)
	InitDelaying(delaying.MustRegisterFunc)
}

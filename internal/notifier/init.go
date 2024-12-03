package notifier

import botgolang "github.com/mail-ru-im/bot-golang"

var bot *botgolang.Bot

func InitNotifier(b *botgolang.Bot) {
	bot = b
}

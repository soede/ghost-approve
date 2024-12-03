package app

import (
	"ghost-approve/internal/handlers"
	"ghost-approve/internal/handlers/callbacks"
	botgolang "github.com/mail-ru-im/bot-golang"
)

func Updates(e *botgolang.EventType, p *botgolang.EventPayload) {
	if *e == botgolang.NEW_MESSAGE {
		handlers.MessageHandler(p)
	} else if *e == botgolang.CALLBACK_QUERY {
		callbacks.CallbackHandler(p)
	}

}

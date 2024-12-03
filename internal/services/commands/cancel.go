package commands

import (
	"ghost-approve/pkg/vkbot"
	botgolang "github.com/mail-ru-im/bot-golang"
	"log"
)

func Cancel(a *botgolang.EventPayload) {
	userState := UserStates[a.From.ID]
	if userState == nil {
		message := vkbot.GetBot().NewTextMessage(a.From.ID, "Для того чтобы отменить апрув, нужно начать его создавать")
		if err := message.Send(); err != nil {
			log.Printf("failed to send message: %s", err)
		}
		return
	}

	message := vkbot.GetBot().NewTextMessage(a.From.ID, "Ты действительно хочешь отменить создание апрува?")

	keyboard := botgolang.NewKeyboard()
	yes := botgolang.NewCallbackButton("Да", "/create_yes_cancel").WithStyle(botgolang.ButtonAttention)
	no := botgolang.NewCallbackButton("Нет", "/create_not_cancel")
	keyboard.AddRow(yes, no)
	message.AttachInlineKeyboard(keyboard)

	if err := message.Send(); err != nil {
		log.Printf("failed to send message: %s", err)
	}

}

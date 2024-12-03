package commands

import (
	"ghost-approve/pkg/vkbot"
	botgolang "github.com/mail-ru-im/bot-golang"
	"log"
)

func Manage(a *botgolang.EventPayload) {
	userID := a.From.ID

	message := vkbot.GetBot().NewMessage(userID)
	text := "Здесь ты можешь посмотреть информацию о созданных апруах. \n" +
		"Нажми на кнопку \"Актуальные\" для того чтобы управлять активными апрувами\n" +
		"Нажми на кнопку \"Отчеты\" чтобы посмотреть отчеты завершенных апрувов"
	message.Text = text
	keyboard := botgolang.NewKeyboard()
	yes := botgolang.NewCallbackButton("Актуальные", "/manage_approves")
	no := botgolang.NewCallbackButton("Отчеты", "/manage_reports")
	keyboard.AddRow(yes, no)
	message.AttachInlineKeyboard(keyboard)
	err := message.Send()
	if err != nil {
		log.Println(err)
		return
	}

}

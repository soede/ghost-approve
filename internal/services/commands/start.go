package commands

import (
	"ghost-approve/internal/repositories"
	"ghost-approve/pkg/vkbot"
	botgolang "github.com/mail-ru-im/bot-golang"
	"log"
)

func Start(a *botgolang.EventPayload) {
	var text string
	var userID = a.From.ID

	user := repositories.GetUserByID(userID)

	if user == nil {
		repositories.CreateUser(userID, a.From.FirstName, a.From.LastName)
		text += "Привет! Меня зовут Споки, я помогу тебе получать апрувы последовательно и эффективно!\n"
	} else if user.Registered {
		text += "Каежтся мы с тобой уже знакомы, меня все еще зовут Споки!👻\n"
	} else if !user.Registered {
		text += "Привет! Меня зовут Споки. Мы еще не знакомы, но тебе уже отправили несколько апрувов, отправь мне /check чтобы их проверить\n"
		err := repositories.ActivateUser(user, a.From.FirstName, a.From.LastName)
		if err != nil {
			log.Println(err)
		}
	}

	text += "Вот основные команды: \n" +
		"/check – проверим все актальные апрувы \n" +
		"/create – создаст новый апрув \n" +
		"/received – управлять полученными апрувами \n" +
		"/manage – управлять созданными апрувами" +
		"\n\nЕще ты можешь получать отчеты по ID. Например, чтобы получить отчет по апруву #1 введи /report1"

	message := vkbot.GetBot().NewTextMessage(userID, text)
	if err := message.Send(); err != nil {
		log.Println(err)
	}
}

package callbacks

import (
	"ghost-approve/internal/services"
	"ghost-approve/internal/services/commands"
	"ghost-approve/internal/utils"
	"ghost-approve/pkg/vkbot"
	botgolang "github.com/mail-ru-im/bot-golang"
	log "github.com/sirupsen/logrus"
	"strconv"
	"strings"
)

func CreateCallback(p *botgolang.EventPayload) {
	userID := p.From.ID
	data := p.CallbackQuery().CallbackData

	if strings.HasPrefix(data, "/create_time_") {
		time, err := strconv.Atoi(strings.TrimPrefix(data, "/create_time_"))
		if err != nil {
			return
		}
		if time < 4 || time > utils.Month {
			return
		}
		setConfirmTime(userID, time, commands.ConfirmTime)
		return
	}

	switch data {
	case "/create_skip":
		if !checkUserStage(userID, commands.Description) {
			return
		}
		commands.UserStates[userID].Description = ""
		commands.Create(p)

	case "/create_private":
		if !checkUserStage(userID, commands.Visibility) {
			return
		}
		commands.UserStates[userID].IsPrivate = true

		if commands.UserStates[userID].EditMode == true {
			commands.UserStates[userID].StopEditAndSendExample(userID)
			return
		}

		commands.UserStates[userID].CurrentStage = commands.Cancellable
		commands.Create(p)

	case "/create_public":
		if !checkUserStage(userID, commands.Visibility) {
			return
		}

		commands.UserStates[userID].IsPrivate = false

		if commands.UserStates[userID].EditMode == true {
			commands.UserStates[userID].StopEditAndSendExample(userID)
			return
		}

		commands.UserStates[userID].CurrentStage = commands.Cancellable
		commands.Create(p)

	case "/create_cancellable":
		if !checkUserStage(userID, commands.Cancellable) {
			return
		}
		commands.UserStates[userID].Cancelable = true
		commands.UserStates[userID].CurrentStage = commands.StopOnReject
		commands.Create(p)

	case "/create_confirmable":
		if !checkUserStage(userID, commands.Cancellable) {
			return
		}
		commands.UserStates[userID].Cancelable = false

		if commands.UserStates[userID].EditMode == true {
			commands.UserStates[userID].StopEditAndSendExample(userID)
			return
		}

		commands.UserStates[userID].CurrentStage = commands.ConfirmTime
		commands.Create(p)

	case "/create_continueOnReject":
		if !checkUserStage(userID, commands.StopOnReject) {
			return
		}
		commands.UserStates[userID].StopOnReject = false

		if commands.UserStates[userID].EditMode == true {
			commands.UserStates[userID].StopEditAndSendExample(userID)
			return
		}

		commands.UserStates[userID].CurrentStage = commands.ConfirmTime
		commands.Create(p)

	case "/create_stopOnReject":
		if !checkUserStage(userID, commands.StopOnReject) {
			return
		}
		commands.UserStates[userID].StopOnReject = true

		if commands.UserStates[userID].EditMode == true {
			commands.UserStates[userID].StopEditAndSendExample(userID)
			return
		}

		commands.UserStates[userID].CurrentStage = commands.ConfirmTime
		commands.Create(p)

	case "/create_other_time":
		if !checkUserStage(userID, commands.ConfirmTime) {
			return
		}
		err := vkbot.GetBot().NewTextMessage(p.From.ID, "Укажи время, в течение которого должен быть подписан апрув\n"+
			"Используй сокращения: ч — часы, д — дни, н — недели, м - месяцы\n"+
			"Пример форматов:\n"+
			"1ч — 1 час\n2д 12ч — 2 дня и 12 часов\n3н — 3 недели\n"+
			"Еще ты можешь ввести нужную дату в формате 2024/9/3 17:00 \n"+
			"Важно: введенное время не должно превышать 2 месяцев").Send()
		if err != nil {
			log.Error(err)
		}

		commands.UserStates[p.From.ID].CurrentStage = commands.OtherTime

	case "/create_with_link":
		if !checkUserStage(userID, commands.NeedLink) {
			return
		}
		commands.UserStates[userID].CurrentStage = commands.RequireLink
		commands.Create(p)
		return

	case "/create_without_link":
		if !checkUserStage(userID, commands.NeedLink) {
			return
		}
		if commands.UserStates[userID].EditMode {
			commands.UserStates[userID].Links = nil
			commands.UserStates[userID].StopEditAndSendExample(userID)
			return
		}
		commands.UserStates[userID].CurrentStage = commands.NeedFile
		commands.Create(p)
		return

	case "/create_with_file":
		if !checkUserStage(userID, commands.NeedFile) {
			return
		}
		commands.UserStates[userID].CurrentStage = commands.FileType
		commands.Create(p)

	case "/create_without_file":
		if !checkUserStage(userID, commands.NeedFile) {
			return
		}
		if commands.UserStates[userID].EditMode {
			commands.UserStates[userID].FileID = ""
			commands.UserStates[userID].IsEditableFile = false
			commands.UserStates[userID].StopEditAndSendExample(userID)
			return
		}
		commands.UserStates[userID].CurrentStage = commands.RequireParticipants
		commands.Create(p)

	case "/create_editable_file":
		if !checkUserStage(userID, commands.FileType) {
			return
		}

		commands.UserStates[userID].IsEditableFile = true
		commands.UserStates[userID].CurrentStage = commands.RequireFile
		commands.Create(p)

	case "/create_standard_file":
		if !checkUserStage(userID, commands.FileType) {
			return
		}

		commands.UserStates[userID].IsEditableFile = false
		commands.UserStates[userID].CurrentStage = commands.RequireFile
		commands.Create(p)

	case "/create_participants_ok":
		if !checkComplete(commands.UserStates[userID]) {
			return
		}

		if commands.UserStates[userID].EditMode {
			commands.UserStates[userID].StopEditAndSendExample(userID)
			return
		}

		commands.UserStates[userID].CurrentStage = commands.ExampleMessage
		commands.Create(p)

	case "/create_participants_again":
		if !checkComplete(commands.UserStates[userID]) {
			return
		}
		commands.UserStates[userID].CurrentStage = commands.RequireParticipants
		commands.Create(p)

	case "/create_yes_send":
		if !checkComplete(commands.UserStates[userID]) {
			return
		}
		if commands.UserStates[userID].CurrentStage != commands.ExampleMessage {
			return
		}
		approval, err := services.CheckAndCreateApprove(commands.UserStates[userID])
		if err != nil {
			log.Error(err)
			return
		}
		services.SendApprovalsToParticipants(approval, commands.UserStates[userID])
		commands.UserStates[userID].CurrentStage = commands.End
		commands.Create(p)

	case "/create_not_send":
		if !checkComplete(commands.UserStates[userID]) {
			return
		}
		if _, exist := commands.UserStates[userID]; !exist {
			return //не найден твой апрув
		}
		commands.SendChangeMessage(userID)

	case "/create_yes_cancel":
		delete(commands.UserStates, p.From.ID)
		message := vkbot.GetBot().NewTextMessage(p.From.ID, "Создание апрува было отменено")
		if err := message.Send(); err != nil {
			log.Error(err)
		}

	case "/create_not_cancel":
		message := vkbot.GetBot().NewTextMessage(p.From.ID, "Апрув не был отменён")
		if err := message.Send(); err != nil {
			log.Error(err)
		}

	}

}

// setConfirmTime – универсальная функция для установки времени подтверждения и перехода на следующий этап
func setConfirmTime(userID string, duration int, requiredStage commands.CurrentStage) {
	if checkUserStage(userID, requiredStage) {
		commands.UserStates[userID].ConfirmTime = duration

		if commands.UserStates[userID].EditMode {
			commands.UserStates[userID].StopEditAndSendExample(userID)
			return
		}

		commands.UserStates[userID].CurrentStage = commands.NeedLink
		commands.SendNeedLink(userID)
	}
}

// checkUserStage – проверяет, что пользователь находится на нужном этапе
func checkUserStage(userID string, stage commands.CurrentStage) bool {
	user, exists := commands.UserStates[userID]
	return exists && user.CurrentStage == stage
}

// Проверка того что апрув уже завершен, участники задаются на последнем этапе, если в апруве есть участники, то он уже заполнен
// если не проверять, то можно создавать новые апрувы и нажимать на старые кнопки, создавая пустые апрувы.
func checkComplete(us *commands.UserState) bool {
	if len(us.Participants) == 0 {
		return false
	}
	return true
}

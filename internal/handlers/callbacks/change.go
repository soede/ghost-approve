package callbacks

import (
	"ghost-approve/internal/services/commands"
	botgolang "github.com/mail-ru-im/bot-golang"
)

func handleChangeCallback(p *botgolang.EventPayload) {
	commands.UserStates[p.From.ID].EditMode = true
	userID := p.From.ID
	switch p.CallbackQuery().CallbackData {
	case "/change_title":
		commands.UserStates[userID].CurrentStage = ""
		commands.Create(p)

	case "/change_description":
		commands.UserStates[userID].CurrentStage = commands.Description
		commands.SendRequireDescription(userID)
	case "/change_cancel":
		commands.UserStates[userID].CurrentStage = commands.Cancellable
		commands.SendRequireCancelable(userID)
	case "/change_duration":
		commands.UserStates[userID].CurrentStage = commands.ConfirmTime
		commands.SendRequireConfirmTime(userID)
	case "/change_link":
		commands.UserStates[userID].CurrentStage = commands.NeedLink
		commands.SendNeedLink(userID)
	case "/change_participants":
		commands.UserStates[userID].CurrentStage = commands.Participants
		commands.SendRequireParticipants(userID)
	case "/change_file":
		commands.UserStates[userID].CurrentStage = commands.NeedFile
		commands.SendNeedFile(userID)
	case "/change_visible":
		commands.UserStates[userID].CurrentStage = commands.Visibility
		commands.SendRequirePrivate(userID)
	case "/not_change":
		commands.UserStates[userID].StopEditAndSendExample(userID)
	}
}

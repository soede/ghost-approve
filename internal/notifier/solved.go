package notifier

import (
	"fmt"
	"ghost-approve/internal/repositories"
	"ghost-approve/internal/utils"
	"ghost-approve/pkg/vkbot"
)

func SendConfirmedMessage(approvalID int, userID string) error {
	approval, err := repositories.GetApprovalByID(approvalID)
	if err != nil {
		return err
	}

	if approval.AuthorID == userID {
		return nil
	}

	authorID := approval.AuthorID
	userLink := utils.CreateUserLink([]string{userID})
	if err != nil {
		return err
	}

	text := fmt.Sprintf("✅Пользователь: %s подтвердил(-a) апрув #%d", userLink, approvalID)

	err = bot.NewTextMessage(authorID, text).Send()

	return err

}

// SendRejectedMessage – отправляет создателю уведомление о том что апрув завершен, а участнику о том что он успешно отклонил апрув
func SendRejectedMessage(approvalID int, userID string, stopOnReject bool) error {
	approval, err := repositories.GetApprovalByID(approvalID)

	if approval.AuthorID == userID && stopOnReject {
		err = vkbot.GetBot().NewTextMessage(userID, fmt.Sprintf("❌Апрув #%d был завершен. Отчет по нему можно найти, введя команду /report%d", approvalID, approvalID)).Send()
		return nil
	}
	if err != nil {
		return err
	}
	authorID := approval.AuthorID

	userLink := utils.CreateUserLink([]string{userID})
	if err != nil {
		return err
	}

	if approval.AuthorID != userID {
		text := fmt.Sprintf("❌Пользователь: %s отклонил(-a) апрув #%d", userLink, approvalID)

		if approval.StopOnReject {
			text += fmt.Sprintf("\n Апрув был завершен. Отчет по нему можно найти, введя команду /report%d", approvalID)
		}
		err = bot.NewTextMessage(authorID, text).Send()
	}

	err = vkbot.GetBot().NewTextMessage(userID, fmt.Sprintf("❌Апрув #%d был отклонен", approvalID)).Send()

	return err

}

package notifier

import (
	"fmt"
	"ghost-approve/internal/models"
	"ghost-approve/internal/repositories"
	"ghost-approve/internal/utils"
	"ghost-approve/pkg/botErrors"
	rdb "ghost-approve/pkg/db/redis"
	log "github.com/sirupsen/logrus"
	"strings"
	"time"
)

func SendRemindMessage(task *models.Task) error {
	if task == nil {
		return botErrors.ErrTaskIsNil
	}
	var err error
	approvalID := task.ApproveID
	member := task.Member

	notReactedUsers, err := repositories.GetUsersNotReacted(approvalID)
	if err != nil {
		return err
	}

	notRegistered, err := repositories.CheckNotRegisteredUsers(approvalID)
	if err != nil {
		return err
	}

	approval, err := repositories.GetApprovalByID(approvalID)
	if err != nil {
		return err
	}

	authorID := approval.AuthorID

	if strings.HasPrefix(member, "end:") { // надо вынести в end
		if approval.Status != models.StatusPending {
			return nil
		}
		err = repositories.EditApproveStatus(approval.ID, models.StatusExpired)
		err = repositories.SetCompletedAt(approval.ID)

		if err != nil {
			return err
		}

		usersID := utils.CreateUserLink(notReactedUsers)
		if err != nil {
			return err
		}
		text := fmt.Sprintf("Апрув #%d был завершен. \nУказанное тобой время истекло, но некоторые участники не дали отклика: %s \nОтправь report/%d для того чтобы посмотреть отчет",
			approval.ID,
			usersID, approval.ID)

		if len(notRegistered) != 0 {
			notRegisteredLinks := utils.CreateUserLink(notRegistered)
			text += fmt.Sprintf("Некоторые участники не написали мне и я не смог отправить им апрув: %s", notRegisteredLinks)
		}

		err = rdb.Client().ZRem("approve_notifications", fmt.Sprintf("end:%d", approvalID))
		err = rdb.Client().ZRem("approve_notifications", fmt.Sprintf("half:%d", approvalID))
		err = rdb.Client().ZRem("approve_notifications", fmt.Sprintf("quarter:%d", approvalID))

		err = bot.NewTextMessage(authorID, text).Send()

		return err

	}

	if approval.Status != models.StatusPending {
		return nil
	}
	if err != nil {
		log.Error(err)
	}

	targetTime := approval.CreatedAt.Add(time.Duration(approval.ConfirmTime) * time.Hour)
	currentTime := time.Now()
	timeRemaining := int(targetTime.Sub(currentTime).Minutes())
	timeString := utils.FormatMinutes(timeRemaining)
	timeLeft := "осталось времени: "

	text := fmt.Sprintf("Не забудь подтвердить апрув #%d, %s %s", approvalID, timeLeft, timeString)

	for _, el := range notReactedUsers {
		if err := bot.NewTextMessage(el, text).Send(); err != nil {
			log.Error(err)
		}
	}

	if len(notRegistered) != 0 {
		text = fmt.Sprintf("Некоторые участники апрува #%d все еще не написали мне: %s. Напомни им написать мне, тогда я смогу отправить им твой апрув", approvalID, strings.Join(notRegistered, ", "))
		if err := bot.NewTextMessage(authorID, text).Send(); err != nil {
			log.Error(err)
		}
	}

	return nil

}

package services

import (
	"errors"
	"fmt"
	"ghost-approve/internal/models"
	"ghost-approve/internal/notifier"
	"ghost-approve/internal/repositories"
	"ghost-approve/internal/services/commands"
	"ghost-approve/internal/utils"
	"ghost-approve/pkg/botErrors"
	"ghost-approve/pkg/db/postgres"
	rdb "ghost-approve/pkg/db/redis"
	"ghost-approve/pkg/vkbot"
	"github.com/lib/pq"
	botgolang "github.com/mail-ru-im/bot-golang"
	log "github.com/sirupsen/logrus"
	"strings"
	"sync"
	"time"
)

type FileInfo struct {
	ApproveID int
	Version   int
}

var WaitForFile = make(map[string]*FileInfo)

func CheckAndCreateApprove(userState *commands.UserState) (*models.Approval, error) {
	user := repositories.GetUserByID(userState.AuthorID)
	if user == nil {
		return nil, errors.New("failed get user from DB")
	}
	users, err := repositories.GetOrCreateUsersByID(userState.Participants)
	if err != nil {
		return nil, err
	}

	var file *models.File
	if err != nil {
		return nil, err
	}

	if userState.FileID != "" {
		file = &models.File{
			FileID:         userState.FileID,
			OriginalFileID: userState.FileID,
			UploaderID:     userState.AuthorID,
			AuthorID:       userState.AuthorID,
		}
	}

	approve := &models.Approval{
		AuthorID:      userState.AuthorID,
		Title:         userState.Title,
		Description:   userState.Description,
		Links:         pq.StringArray(userState.Links),
		ConfirmTime:   userState.ConfirmTime,
		Status:        models.StatusPending,
		Cancelable:    userState.Cancelable,
		StopOnReject:  userState.StopOnReject,
		Editable:      userState.IsEditableFile,
		IsPrivate:     userState.IsPrivate,
		Participants:  users,
		TotalComplete: 0,
		File:          file,
	}

	approve, err = repositories.CreateApprove(approve)
	if err != nil {
		return nil, err
	}

	err = repositories.CreateTasks(approve.ID, approve.ConfirmTime)
	if err != nil {
		return nil, err
	}

	if err != nil {
		return nil, err
	}

	return approve, nil

}

// SendApprovalsToParticipants – отправляет апрувы всем участникам
func SendApprovalsToParticipants(approval *models.Approval, us *commands.UserState) {
	var wg sync.WaitGroup

	for _, p := range us.Participants {
		message := commands.ApproveMessage(approval, p)
		message1 := vkbot.GetBot().NewTextMessage(p, "Ты получил новый апрув!")
		go func(msgToSend, msgToSend1 *botgolang.Message) {
			wg.Add(1)
			if err := msgToSend1.Send(); err != nil {
				log.Errorf("failed to send message: %s", err)
			}
			if err := msgToSend.Send(); err != nil {
				log.Errorf("failed to send message: %s", err)
			}
			defer wg.Done()
		}(message, message1)
	}
	wg.Wait()
}

// ConfirmApprove – подтверждает апрув
func ConfirmApprove(approveID, version int, userID string) error {
	var err error
	approval, err := repositories.GetApproveByID(approveID)
	if err != nil || approval == nil {
		return err
	}
	access := ConfirmApprovalAccess(approveID, userID)
	if !access {
		return botErrors.ErrNoAccess
	}
	if approval.Status != models.StatusPending {
		return botErrors.ErrApprovalIsNotRelevant
	}

	if approval.Editable {
		_, exists := WaitForFile[userID]
		if !exists {
			if err != nil {
				return err
			}
			WaitForFile[userID] = &FileInfo{ApproveID: approveID, Version: version}

			SendFileOrCancel(userID, fmt.Sprintf("Для того чтобы подтвердить апрув #%d нужно отправить обновленную версию прикрепленного документа. "+
				"\nОтправь мне его сейчас и я добавлю тебя в список тех, кто подтвердил апрув", approveID))

			return err
		}

	}
	user := repositories.GetUserByID(userID)
	err = repositories.AddUserToApprovedUsers(approveID, user)

	if err != nil {
		return err
	}

	notReacted, err := repositories.GetUsersNotReacted(approveID)
	if err != nil {
		return err
	}

	notReg, err := repositories.CheckNotRegisteredUsers(approveID)

	if err != nil {
		return err
	}
	if err != nil {
		log.Error(err)
	}

	if !approval.Editable {
		err = vkbot.GetBot().NewTextMessage(userID, fmt.Sprintf("Апрув #%d был успешно подтвержден!", approveID)).Send()
	}

	err = notifier.SendConfirmedMessage(approveID, userID)

	if len(notReacted) == 0 && len(notReg) == 0 {
		status, err := changeApproveStatus(approval.ID)
		if err != nil {
			return err
		}
		err = repositories.SetCompletedAt(approveID)
		if err != nil {
			return err
		}
		text := fmt.Sprintf("Апрув #%d был заврешён и получил статус \"%s\". Чтобы посмотреть отчет отправь /report%d", approval.ID, status, approval.ID)
		SendFinish(approval.AuthorID, approval.ID, text)
	}

	return nil
}

func RejectApprove(approveID int, userID string) error {
	var err error

	approval, err := repositories.GetApproveByID(approveID)
	if err != nil || approval == nil {
		return err
	}

	access := ConfirmApprovalAccess(approveID, userID)
	if !access {
		return botErrors.ErrNoAccess
	}
	if approval.Status != models.StatusPending {
		return botErrors.ErrApprovalIsNotRelevant
	}

	user := repositories.GetUserByID(userID)
	if err != nil {
		return err
	}

	err = repositories.AddUserToRejectedBy(approveID, user)
	if err != nil {
		return err
	}

	notReacted, err := repositories.GetUsersNotReacted(approveID)
	if err != nil {
		return err
	}

	notReg, err := repositories.CheckNotRegisteredUsers(approveID)

	if err != nil {
		return err
	}

	if approval.StopOnReject {
		err := repositories.EditApproveStatus(approveID, models.StatusRejected)
		err = repositories.SetCompletedAt(approveID)
		if err != nil {
			return err
		}
		err = rdb.Client().ZRem("approve_notifications", fmt.Sprintf("end:%d", approveID))
		err = rdb.Client().ZRem("approve_notifications", fmt.Sprintf("half:%d", approveID))
		err = rdb.Client().ZRem("approve_notifications", fmt.Sprintf("quarter:%d", approveID))

		if err != nil {
			return err
		}

	}

	if err != nil {
		return err
	}
	err = notifier.SendRejectedMessage(approveID, userID, approval.StopOnReject)

	if len(notReacted) == 0 && len(notReg) == 0 && !approval.StopOnReject {
		status, err := changeApproveStatus(approveID)
		if err != nil {
			return err
		}
		err = repositories.SetCompletedAt(approveID)
		if err != nil {
			return err
		}
		SendFinish(approval.AuthorID,
			approval.ID,
			fmt.Sprintf("Апрув #%d был завeршён и получил статус \"%s\" Чтобы посмотреть отчет отправь /report_%d", approval.ID, status, approval.ID))
	}

	if err != nil {
		return err
	}
	return nil
}

func changeApproveStatus(approvalID int) (models.ApproveStatus, error) {
	var rejectedCount, approvedCount int

	err := postgres.GetDB().Raw(`
		SELECT COUNT(*)
		FROM rejected_users
		WHERE approve_id = ?`, approvalID).Scan(&rejectedCount).Error
	if err != nil {
		return "", err
	}

	err = postgres.GetDB().Raw(`
		SELECT COUNT(*)
		FROM approved_users
		WHERE approve_id = ?`, approvalID).Scan(&approvedCount).Error
	if err != nil {
		return "", err
	}

	var newStatus models.ApproveStatus
	if rejectedCount > 0 && approvedCount > 0 {
		newStatus = models.StatusPartiallyApproved
	}

	if rejectedCount > 0 && approvedCount == 0 {
		newStatus = models.StatusRejected
	}

	if approvedCount > 0 && rejectedCount == 0 {
		newStatus = models.StatusApproved
	}

	if newStatus != "" {
		err = postgres.GetDB().Model(&models.Approval{}).
			Where("id = ?", approvalID).
			Update("status", newStatus).Error
		if err != nil {
			return "", err
		}
	}

	return newStatus, nil
}

func SendFinish(authorID string, approveID int, text string) {
	err := rdb.Client().ZRem("approve_notifications", fmt.Sprintf("end:%d", approveID))
	err = rdb.Client().ZRem("approve_notifications", fmt.Sprintf("half:%d", approveID))
	err = rdb.Client().ZRem("approve_notifications", fmt.Sprintf("quarter:%d", approveID))

	if err != nil {
		log.Error(err)
	}
	if err := vkbot.GetBot().NewTextMessage(authorID, text).Send(); err != nil {
		log.Error(err)
	}
}

// SendManageApprovals – открывает панель управления апрувами
func SendManageApprovals(userID string) {
	approvals, err := repositories.ManageApprovals(userID)
	if err != nil {
		log.Error(err)
		return
	}

	if len(approvals) == 0 {
		err := vkbot.GetBot().NewTextMessage(userID, "У тебя пока что нет созданных актуальных апрувов\n"+
			"Но ты можешь создать их отправив мне команду /create").Send()
		if err != nil {
			log.Error(err)
		}
		return
	}

	for _, el := range approvals {
		go func(approve models.Approval) {
			message := ManageApproveMessage(&approve, userID)
			//
			keyboard := botgolang.NewKeyboard()
			statistic := botgolang.NewCallbackButton("📊Статистика", fmt.Sprintf("/manage_statistic_%d", approve.ID))
			events := botgolang.NewCallbackButton("🕒 История действий", fmt.Sprintf("/manage_events_%d", approve.ID))
			notification := botgolang.NewCallbackButton("🔔Отправить напоминание", fmt.Sprintf("/manage_notification_%d", approve.ID))
			cancel := botgolang.NewCallbackButton("❌Отменить апрув", fmt.Sprintf("/manage_cancel_%d", approve.ID))
			keyboard.AddRow(statistic, events)
			keyboard.AddRow(notification, cancel)

			message.AttachInlineKeyboard(keyboard)
			//
			if err := message.Send(); err != nil {
				log.Errorf("Ошибка отправки сообщения:", err)
			}
		}(el)
	}

}

// FetchStats – собирает статистику по апруву в сообщение
func FetchStats(approveID int) string {
	var text string

	text += "👥Пользователи\n"

	approved, err := repositories.ApprovedUsersID(approveID)
	if err != nil {
		log.Error(err)
	}
	approvedLinks := utils.CreateUserLink(approved)
	if err != nil {
		log.Error(err)
	}

	if len(approvedLinks) != 0 {
		text += fmt.Sprintf("Подтвердили апрув: %s \n", approvedLinks)
	}

	rejected, err := repositories.RejectedUsersID(approveID)
	if err != nil {
		log.Error(err)
	}
	rejectedLinks := utils.CreateUserLink(rejected)

	if len(rejectedLinks) != 0 {
		text += fmt.Sprintf("Отклонили апрув: %s \n", rejectedLinks)

	}

	notReacted, err := repositories.GetUsersNotReacted(approveID)
	if err != nil {
		log.Error(err)
	}
	notReactedLinks := utils.CreateUserLink(notReacted)
	if len(notReacted) != 0 {
		text += fmt.Sprintf("Не дали свой отклик: %s \n", notReactedLinks)
	}

	notReg, err := repositories.CheckNotRegisteredUsers(approveID)
	if err != nil {
		log.Error(err)
	}
	if len(notReg) != 0 {
		text += fmt.Sprintf("Еще не написали мне: %s \n", strings.Join(notReg, ", "))
	}

	reminds, err := repositories.CountApprovalReminders(approveID)
	if err != nil {
		log.Error(err)
	}

	text += "🔔Уведомления\n"

	text += fmt.Sprintf("Количество отправленных напоминаний: %d \n", reminds)

	taskTime := repositories.NextApprovalTask(approveID)

	if taskTime == -1 {
		text += fmt.Sprintf("Я уже отправил все автоматические напоминания участникам\n\n")
	}
	if taskTime != -1 {
		now := time.Now().Unix()
		diffInSeconds := int64(taskTime) - now
		minutes := diffInSeconds / 60
		text += fmt.Sprintf("Следующее автоматическое напоминание будет отправлено через %s \n\n", utils.FormatMinutes(int(minutes)))
	}
	text += "Автоматические напоминания отправляются после того, как пройдет половина времени или останется четверть времени"

	if err != nil {
		log.Error(err)
	}

	return text

}

type EventElement struct {
	Text    string
	Created time.Time
}

func CancelApprove(approveID int) error {
	err := repositories.SetCompletedAt(approveID)
	err = repositories.EditApproveStatus(approveID, models.StatusCanceled)
	err = rdb.Client().ZRem("approve_notifications", fmt.Sprintf("end:%d", approveID))
	err = rdb.Client().ZRem("approve_notifications", fmt.Sprintf("half:%d", approveID))
	err = rdb.Client().ZRem("approve_notifications", fmt.Sprintf("quarter:%d", approveID))
	return err
}

// IsCurrentApprove – проверяет что апрув актуален
func IsCurrentApprove(approveID int) (bool, error) {
	approval, err := repositories.GetApprovalByID(approveID)
	if err != nil {
		return false, err
	}

	return approval.Status == models.StatusPending, nil
}

func ManageApproveMessage(approve *models.Approval, userID string) *botgolang.Message {
	exampleMessage := vkbot.GetBot().NewMessage(userID)
	var file *models.File
	result := postgres.GetDB().Where("approve_id = ?", approve.ID).First(&file)
	if result != nil && result.Error == nil {
		exampleMessage.FileID = file.FileID
	}

	exampleMessage.Text = ManageApproveText(approve)
	keyboard := botgolang.NewKeyboard()

	exampleMessage.AttachInlineKeyboard(keyboard)
	return exampleMessage

}

func ManageApproveText(approve *models.Approval) string {
	var exampleText string
	exampleText += fmt.Sprintf("Апрув #%d", approve.ID)
	exampleText += fmt.Sprintf("\n%s", approve.Title)

	if len(approve.Description) != 0 {
		exampleText += "\nОписание: " + approve.Description
	}
	authorLink := utils.CreateUserLink([]string{approve.AuthorID})
	exampleText += fmt.Sprintf("\nОт: %s", authorLink)

	months := []string{
		"янв", "фев", "мар", "апр", "май", "июн",
		"июл", "авг", "сен", "окт", "ноя", "дек",
	}
	ca := approve.CreatedAt
	exampleText += fmt.Sprintf("\nБыло отправлено: %02d %s %02d:%02d",
		ca.Day(), months[ca.Month()-1], ca.Hour(), ca.Minute())

	targetTime := approve.CreatedAt.Add(time.Duration(approve.ConfirmTime) * time.Hour)
	currentTime := time.Now()
	timeRemaining := int(targetTime.Sub(currentTime).Minutes())
	if timeRemaining >= 0 && approve.Status == models.StatusPending {
		exampleText += "\nОсталось времени: " + utils.FormatMinutes(timeRemaining)
	}
	if timeRemaining < 0 || approve.Status != models.StatusPending {
		exampleText += fmt.Sprintf("\nСтатус: %s", approve.Status)
	}

	if len(approve.Links) != 0 {
		exampleText += "\nПрикрепленные ссылки: "

		exampleText += strings.Join(approve.Links, ", ")
	}

	if approve.Cancelable {
		exampleText += "\nТип апрува: отклоняемый"
		if approve.StopOnReject {
			exampleText += ", завершается после первого отклонения"
		}
		if !approve.StopOnReject {
			exampleText += ", но не завершается после первого отклонения"
		}
	}
	if !approve.Cancelable {
		exampleText += "\nТип апрува: не отклоняемый"

	}

	if approve.Editable {
		exampleText += "\nТип файла: редактируемый, каждый участник должен отправить его обновленную версию"
	}

	return exampleText

}

func ConfirmApprovalAccess(approvalID int, userID string) bool {
	users, err := repositories.GetParticipantsByApprovalID(approvalID)
	if err != nil {
		return false
	}

	for _, user := range users {
		if user.ID == userID {
			return true
		}
	}
	return false
}

package services

import (
	"errors"
	"fmt"
	"ghost-approve/internal/models"
	"ghost-approve/internal/repositories"
	"ghost-approve/internal/utils"
	"ghost-approve/pkg/botErrors"
	"ghost-approve/pkg/db/postgres"
	"ghost-approve/pkg/vkbot"
	botgolang "github.com/mail-ru-im/bot-golang"
	log "github.com/sirupsen/logrus"
	"gorm.io/gorm"
	"sort"
	"strings"
	"time"
)

// ReportsApprovalsByUserID – вернет апрувы из которых можно создать отчет
func ReportsApprovalsByUserID(authorID string) ([]*botgolang.Message, error) {
	var messages []*botgolang.Message
	approvals, err := repositories.EndApprovalsByUserID(authorID)
	if errors.Is(err, botErrors.NotFoundApprovalsForReports) {
		messages = append(messages, vkbot.GetBot().NewTextMessage(authorID, "Не найдено заврешенных апрувов для генерации отчетов"))
		return messages, nil
	}
	if err != nil {
		log.Error(err)
		return nil, err
	}

	for _, approval := range *approvals {
		message := vkbot.GetBot().NewTextMessage(authorID, generateEndApprovalText(approval.ID, approval.Title, approval.Status, approval.CreatedAt))
		keyboard := botgolang.NewKeyboard()
		getReport := botgolang.NewCallbackButton("📄Получить отчет", fmt.Sprintf("/manage_report_%d", approval.ID))
		deleteReport := botgolang.NewCallbackButton("🗑️Удалить отчет", fmt.Sprintf("/manage_ask_dreport_%d", approval.ID)).WithStyle(botgolang.ButtonAttention)
		keyboard.AddRow(getReport, deleteReport)
		message.AttachInlineKeyboard(keyboard)

		messages = append(messages, message)
	}

	return messages, nil

}

// вернете
func ReportMessage(approvalID int, authorID string) (message *botgolang.Message, err error) {
	message = vkbot.GetBot().NewMessage(authorID)
	approval, err := repositories.GetApprovalByID(approvalID)
	events, err := FetchEvents(approvalID)
	if err != nil {
		return nil, err
	}
	stats, err := CollectStats(approvalID)
	if err != nil {
		return nil, err
	}
	message.Text += fmt.Sprintf("Отчет по апруву #%d\n", approvalID)
	message.Text += "Основная информация\n"
	message.Text += ReportApprovalText(approval) + "\n\n"
	message.Text += stats
	message.Text += "\nИстория действий\n"
	message.Text += events

	return

}

func generateEndApprovalText(ID int, title string, status models.ApproveStatus, createdAt time.Time) (text string) {
	text += fmt.Sprintf("Апрув #%d\n", ID)
	text += fmt.Sprintf("Заголовок: %s\n", title)

	months := []string{
		"янв", "фев", "мар", "апр", "май", "июн",
		"июл", "авг", "сен", "окт", "ноя", "дек",
	}
	ca := createdAt
	text += fmt.Sprintf("\nБыло отправлено: %02d %s %02d:%02d\n",
		ca.Day(), months[ca.Month()-1], ca.Hour(), ca.Minute())

	text += fmt.Sprintf("Статус: %s", status)
	return
}

func FormatReport(events []EventElement) string {
	if len(events) == 0 {
		return ""
	}

	sort.Slice(events, func(i, j int) bool {
		return events[i].Created.Before(events[j].Created)
	})

	var result string
	var currentDate string

	for _, event := range events {
		month := utils.Months[event.Created.Month()]
		dateKey := fmt.Sprintf("%dгод, %02d %s", event.Created.Year(), event.Created.Day(), month)

		if dateKey != currentDate {
			if currentDate != "" {
				result += "\n"
			}
			currentDate = dateKey
			result += dateKey + ":\n"
		}

		eventTime := event.Created.Format("15:04")
		result += fmt.Sprintf("    %s – %s\n", eventTime, event.Text)
	}

	return result
}

// FetchEvents – собирает ивенты
func FetchEvents(approveID int) (string, error) {
	var err error
	var text string

	approval, err := repositories.GetApprovalByID(approveID)
	if errors.Is(gorm.ErrRecordNotFound, err) || approval == nil {
		return "", gorm.ErrRecordNotFound

	}
	if err != nil {
		return "", err
	}
	file, err := repositories.FileByApprovalID(approveID)
	var fileURL string
	if file != nil && err == nil {
		fileURL, err = utils.FileUrlByID(file.OriginalFileID)
		if err != nil {
			log.Error(err)
		}
	}

	if err != nil {
		log.Error(err)
	}
	var events []EventElement

	if err != nil {
		return "", errors.New("Апрув не найден")
	}
	if err != nil {
		log.Error(err)
	}
	if approval.Editable && file != nil {
		events = append(events, EventElement{
			Created: approval.CreatedAt,
			Text:    fmt.Sprintf("📄Пользователь %s создал апрув с редактируемым файлом \n        🔗Ссылка на файл: %s", utils.CreateUserLink([]string{approval.AuthorID}), fileURL)})
	}
	if !approval.Editable && file != nil {
		events = append(events, EventElement{
			Created: approval.CreatedAt,
			Text:    fmt.Sprintf("📄Пользователь %s создал апрув с файлом \n        🔗Ссылка на файл: %s", utils.CreateUserLink([]string{approval.AuthorID}), fileURL)})
	}
	if !approval.Editable && file == nil {
		events = append(events, EventElement{
			Created: approval.CreatedAt,
			Text:    fmt.Sprintf("📄Пользователь %s создал апрув", utils.CreateUserLink([]string{approval.AuthorID}))})
	}

	if approval.Editable {
		approved, err := repositories.GetFileHistoriesByID(approveID)
		if err != nil {
			log.Error(err)
		}

		for _, el := range *approved {
			url, err := utils.FileUrlByID(el.FileID)
			if err != nil {
				log.Error(err)
			}

			events = append(events, EventElement{
				Created: el.UploadedAt,
				Text:    fmt.Sprintf("📝Пользователь %s подтвердил апрув и загрузил обновлённый файл (версия %d) \n        🔗Ссылка на файл: %s", utils.CreateUserLink([]string{el.UploaderID}), el.Version, url)})
		}
	}

	if !approval.Editable {
		approved, err := repositories.ApprovedUsers(approveID)
		if err != nil {
			log.Error(err)
		}

		for _, el := range *approved {
			events = append(events, EventElement{
				Created: el.CreatedAt,
				Text:    fmt.Sprintf("✅Пользователь %s подтвердил апрув", utils.CreateUserLink([]string{el.UserID}))})
		}
	}

	rejected, err := repositories.RejectedUsers(approveID)
	for _, el := range *rejected {
		events = append(events, EventElement{
			Created: el.CreatedAt,
			Text:    fmt.Sprintf("❌Пользователь %s отклонил апрув", utils.CreateUserLink([]string{el.UserID}))})
	}

	reminders, err := repositories.ApprovalRemindersByID(approveID)

	for _, el := range *reminders {
		events = append(events, EventElement{
			Created: el.CreatedAt,
			Text:    fmt.Sprintf("🔔Пользователь %s отправил напоминание всем, кто не дал отклик", utils.CreateUserLink([]string{approval.AuthorID}))})
	}

	now := approval.CreatedAt
	halfTime := now.Add(time.Duration(approval.ConfirmTime/2) * time.Hour)
	quarterTime := now.Add(time.Duration(approval.ConfirmTime-approval.ConfirmTime/4) * time.Hour)
	if halfTime.Before(now) {
		events = append(events, EventElement{
			Text:    "🔔Бот напомнил о том, что осталось меньше половины времени всем кто не дал отклик",
			Created: halfTime,
		})
	}
	if quarterTime.Before(now) {
		events = append(events, EventElement{
			Text:    "🔔Бот напомнил о том, что осталось меньше четверти времени всем, кто не дал отклик",
			Created: quarterTime,
		})
	}

	if !approval.CompletedAt.IsZero() {
		events = append(events, EventElement{
			Text:    "🏁Апрув был завершен",
			Created: approval.CompletedAt,
		})
	}

	text += FormatReport(events)
	return text, nil

}
func CollectStats(approveID int) (string, error) {
	var text string

	text += "👥Пользователи\n"

	approved, err := repositories.ApprovedUsersID(approveID)
	if err != nil {
		return "", err
	}
	approvedLinks := utils.CreateUserLink(approved)
	if err != nil {
		return "", err
	}

	if len(approvedLinks) != 0 {
		text += fmt.Sprintf("Подтвердили апрув: %s \n", approvedLinks)
	}

	rejected, err := repositories.RejectedUsersID(approveID)
	if err != nil {
		return "", err
	}
	rejectedLinks := utils.CreateUserLink(rejected)

	if len(rejectedLinks) != 0 {
		text += fmt.Sprintf("Отклонили апрув: %s \n", rejectedLinks)

	}

	notReacted, err := repositories.GetUsersNotReacted(approveID)
	if err != nil {
		return "", err
	}
	notReactedLinks := utils.CreateUserLink(notReacted)
	if len(notReacted) != 0 {
		text += fmt.Sprintf("Не дали свой отклик: %s \n", notReactedLinks)
	}

	notReg, err := repositories.CheckNotRegisteredUsers(approveID)
	if err != nil {
		return "", err
	}
	if len(notReg) != 0 {
		text += fmt.Sprintf("Еще не написали мне: %s \n", strings.Join(notReg, ", "))
	}

	reminds, err := repositories.CountApprovalReminders(approveID)
	if err != nil {
		return "", err
	}

	text += "🔔Уведомления\n"

	text += fmt.Sprintf("Количество отправленных напоминаний: %d \n", reminds)
	if err != nil {
		return "", err
	}

	return text, nil

}

func CheckApprovalAccess(approvalID int, userID string) (bool, error) {
	approval, err := repositories.GetApprovalByID(approvalID)
	if err != nil {
		return false, err
	}

	if approval.AuthorID == userID {
		return true, nil
	}

	if approval.IsPrivate {
		return false, nil
	}

	err = postgres.GetDB().Preload("Participants").
		Where("id = ? AND is_private = ?", approvalID, false).
		First(&approval).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return false, nil
		}
		return false, err
	}
	for _, participant := range approval.Participants {
		if participant.ID == userID {
			return true, nil
		}
	}
	return false, nil

}

// Собирает текст для отчета
func ReportApprovalText(approve *models.Approval) string {
	var exampleText string
	exampleText += fmt.Sprintf("%s", approve.Title)

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

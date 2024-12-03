package commands

import (
	"errors"
	"fmt"
	"ghost-approve/internal/repositories"
	utlis "ghost-approve/internal/utils"
	"ghost-approve/pkg/botErrors"
	"ghost-approve/pkg/vkbot"
	botgolang "github.com/mail-ru-im/bot-golang"
	"log"
	"os"
	"strings"
	"time"
	"unicode/utf8"
)

type CurrentStage string

const (
	RequireTitle        CurrentStage = "RequireTitle"
	Title               CurrentStage = "Title"
	Description         CurrentStage = "Description"
	Visibility          CurrentStage = "Visibility"
	Cancellable         CurrentStage = "Cancellable"
	StopOnReject        CurrentStage = "StopOnReject"
	ConfirmTime         CurrentStage = "ConfirmTime"
	OtherTime           CurrentStage = "OtherTime"
	CheckTime           CurrentStage = "CheckTime"
	NeedLink            CurrentStage = "NeedLink"
	RequireLink         CurrentStage = "RequireLink"
	CheckLink           CurrentStage = "CheckLink"
	NeedFile            CurrentStage = "AskFile"
	FileType            CurrentStage = "FileType"
	RequireFile         CurrentStage = "RequireFile"
	CheckFile           CurrentStage = "CheckFile"
	RequireParticipants CurrentStage = "RequireParticipants"
	Participants        CurrentStage = "Participants"
	ExampleMessage      CurrentStage = "ExampleMessage"
	End                 CurrentStage = "End"
)

type UserState struct {
	CurrentStage
	EditMode bool // for func (находится если пользователь в режиме редактирования,
	// если да, то функция в callback не пустит его дальше текущего поля)
	// нужно очень хорошо покрыть тестами этот участок
	MessageID      string //for func
	AuthorID       string
	Title          string
	Description    string
	Links          []string
	ConfirmTime    int
	Cancelable     bool
	StopOnReject   bool
	IsEditableFile bool
	IsPrivate      bool
	Participants   []string
	NotRegistered  []string //for func
	FileID         string
}

var UserStates = make(map[string]*UserState)

func Create(a *botgolang.EventPayload) {
	userState := UserStates[a.From.ID]
	userID := a.From.ID
	if userState == nil {
		if user := repositories.GetUserByID(userID); user == nil {
			repositories.CreateUser(userID, a.From.FirstName, a.From.LastName)
		}
		userState = &UserState{}
		userState.AuthorID = a.From.ID
		UserStates[userID] = userState
	}

	switch userState.CurrentStage {

	case Title:
		if utf8.RuneCountInString(a.Text) == 0 {
			err := vkbot.GetBot().NewTextMessage(userID, "Заголовок не может быть пустым").Send()
			if err != nil {
				return
			}
			return
		}

		if utf8.RuneCountInString(a.Text) > 500 {
			err := vkbot.GetBot().NewTextMessage(userID, "Заголовок не может быть длинее 500 символов").Send()
			if err != nil {
				return
			}
			return
		}

		userState.Title = a.Text
		if userState.EditMode == true {
			userState.StopEditAndSendExample(userID)
			return
		}

		SendRequireDescription(userID)
		userState.CurrentStage = Description

	case Description:
		if utf8.RuneCountInString(a.Text) > 1000 {
			err := vkbot.GetBot().NewTextMessage(userID, "Описание не может быть длиннее 1000 символов").Send()
			if err != nil {
				return
			}
			return
		}
		userState.Description = a.Text
		if userState.EditMode == true {
			userState.StopEditAndSendExample(userID)
			return
		}
		userState.CurrentStage = Visibility
		SendRequirePrivate(userID)
	case Visibility:
		SendRequirePrivate(userID)
	case Cancellable:
		SendRequireCancelable(userID)
	case StopOnReject:
		SendRequireStopOnReject(userID)
	case ConfirmTime:
		SendRequireConfirmTime(userID)
		//userState.CurrentStage = CheckTime
	case OtherTime:
		ok := userState.ParseDuration(a.Text, userID)
		if !ok {
			vkbot.GetBot().NewTextMessage(userID, "Я не смог понять что ты имел ввиду, попробуй еще раз")
			return
		}
		if userState.EditMode {
			userState.StopEditAndSendExample(userID)
			return
		}
		userState.CurrentStage = NeedLink
		SendNeedLink(userID)
	case CheckTime:
		if userState.ConfirmTime == 0 || userState.EditMode {
			ok := userState.ParseDuration(a.Text, userID)
			if !ok {
				userState.CurrentStage = CheckTime
				SendRequireConfirmTime(userID)
				log.Println("d")
				return
			}
			if userState.EditMode {
				userState.StopEditAndSendExample(userID)
				return
			}
			userState.CurrentStage = NeedLink
			SendNeedLink(userID)
		}

	case NeedLink:
		SendNeedLink(userID)
	case RequireLink:
		userState.SendRequireForLink(userID)
		userState.CurrentStage = CheckLink
	case CheckLink:
		links, ok := CheckLinks(a.Text, userID)
		if ok {
			userState.SetLinks(userID, links)
			return
		}
	case NeedFile:
		SendNeedFile(userID)
	case FileType:
		SelectFileType(userID)
	case RequireFile:
		AskFile(userID)
		userState.CurrentStage = CheckFile
	case CheckFile:
		ok := userState.SetFile(a)
		if ok {
			if userState.EditMode {
				userState.StopEditAndSendExample(userID)
				return
			}
			SendRequireParticipants(userID)
			userState.CurrentStage = Participants
		}
		if !ok {
			userState.CurrentStage = NeedFile
			SendNeedFile(userID) //если файл не удалось распознать
		}
	case RequireParticipants:
		SendRequireParticipants(userID)
		userState.CurrentStage = Participants
	case Participants:
		userState.SetParticipants(a.Text, a.From.ID)
	case ExampleMessage:
		userState.SendExampleMessage(userID)
		sendFinalOkMessage(userID)
	case End:
		SendApproveCreatedMessage(userID)
	default:
		userState.CurrentStage = Title
		if !userState.EditMode {
			startMessage := vkbot.GetBot().NewTextMessage(userID, "Отлично, давай приступим к созданию апрува!\n"+
				"Если нужно будет отменить создание, отправь мне /cancel в любой момент создания")
			if err := startMessage.Send(); err != nil {
				log.Printf("failed to send message: %s", err)
			}
		}

		message := vkbot.GetBot().NewTextMessage(userID, "Напиши мне заголовок для апрува")
		if err := message.Send(); err != nil {
			log.Printf("failed to send message: %s", err)
		}
	}
}

func (us *UserState) generateApproveText() string {
	var exampleText string
	exampleText += us.Title

	if len(us.Description) != 0 {
		exampleText += "\nОписание: " + us.Description
	}

	userLink := utlis.CreateUserLink([]string{us.AuthorID})
	exampleText += fmt.Sprintf("\nОт: %s", userLink)

	months := []string{
		"янв", "фев", "мар", "апр", "май", "июн",
		"июл", "авг", "сен", "окт", "ноя", "дек",
	}

	now := time.Now()
	exampleText += fmt.Sprintf("\nБыло отправлено: %02d %s %02d:%02d",
		now.Day(), months[now.Month()-1], now.Hour(), now.Minute())
	targetTime := now.Add(time.Duration(us.ConfirmTime) * time.Hour)
	currentTime := time.Now()
	timeRemaining := int(targetTime.Sub(currentTime).Minutes())
	exampleText += "\nОсталось времени: " + utlis.FormatMinutes(timeRemaining)

	if us.IsEditableFile {
		exampleText += "\n\nОтправь отредактированную версию документа для подтверждения. Когда будешь готов, нажми \"подтвердить\""
	}
	return exampleText

}
func SendRequireDescription(userID string) {
	message := vkbot.GetBot().NewTextMessage(userID, "Теперь напиши описание для заявки")
	noButton := botgolang.NewCallbackButton("Пропустить", "/create_skip")
	keyboard := botgolang.NewKeyboard()
	keyboard.AddRow(noButton)
	message.AttachInlineKeyboard(keyboard)

	if err := message.Send(); err != nil {
		log.Printf("failed to send message: %s", err)
	}
}
func SendRequireCancelable(userID string) {
	message := vkbot.GetBot().NewTextMessage(userID, "Этот апрув можно будет отклонять?\n"+
		"\nАпрув с возможностью отклонения будет иметь кнопки \"Подтвердить\" и \"Отклонить\", а также покажет, кто из участников подтвердил, а кто отклонил, в отличие от обычного апрува, где только \"Подтвердить\"")
	yesButton := botgolang.NewCallbackButton("Да", "/create_cancellable")
	noButton := botgolang.NewCallbackButton("Нет", "/create_confirmable")
	keyboard := botgolang.NewKeyboard()
	keyboard.AddRow(yesButton, noButton)
	message.AttachInlineKeyboard(keyboard)

	if err := message.Send(); err != nil {
		log.Printf("failed to send message: %s", err)
	}
}

func SendRequirePrivate(userID string) {
	message := vkbot.GetBot().NewTextMessage(userID, "Этот апрув будет публичным или приватным?\n"+
		"Публичный означает, что все участники смогут просматривать отчёты, статистику и результаты\n"+
		"Приватный даёт доступ к этой информации только тебе")
	private := botgolang.NewCallbackButton("Приватным", "/create_private")
	public := botgolang.NewCallbackButton("Публичным", "/create_public")
	keyboard := botgolang.NewKeyboard()
	keyboard.AddRow(private, public)
	message.AttachInlineKeyboard(keyboard)

	if err := message.Send(); err != nil {
		log.Printf("failed to send message: %s", err)
	}
}
func SendRequireStopOnReject(userID string) {
	message := vkbot.GetBot().NewTextMessage(userID, "Что делать, если кто-то отклонит апрув?\n"+
		"• Продолжить — финальный отчет после всех одобрений/отклонений \n"+
		"• Остановиться — я пришлю отчет с именем того, кто отклонил, и списком тех, кто успел одобрить")
	yesButton := botgolang.NewCallbackButton("Продолжить", "/create_continueOnReject")
	noButton := botgolang.NewCallbackButton("Остановиться", "/create_stopOnReject")
	keyboard := botgolang.NewKeyboard()
	keyboard.AddRow(yesButton, noButton)
	message.AttachInlineKeyboard(keyboard)

	if err := message.Send(); err != nil {
		log.Printf("failed to send message: %s", err)
	}
}
func SendRequireConfirmTime(userID string) {
	message := vkbot.GetBot().NewTextMessage(userID, "В течение какого времени должен быть подписан апрув? \n"+
		"Если тут нет подходящего варианта нажми на кнопку \"Другое время\":\n")

	four := botgolang.NewCallbackButton("4 часа", "/create_time_4")
	day := botgolang.NewCallbackButton("1 день", "/create_time_24")
	three := botgolang.NewCallbackButton("3 дня", "/create_time_72")
	week := botgolang.NewCallbackButton("Неделя", "/create_time_168")
	without := botgolang.NewCallbackButton("Месяц", "/create_time_720")
	other := botgolang.NewCallbackButton("Другое время", "/create_other_time")
	keyboard := botgolang.NewKeyboard()
	keyboard.AddRow(four, day, three, week, without)
	keyboard.AddRow(other)
	message.AttachInlineKeyboard(keyboard)

	if err := message.Send(); err != nil {
		log.Printf("failed to send message: %s", err)
	}
}
func SendNeedLink(userID string) {
	message := vkbot.GetBot().NewTextMessage(userID, "Ты бы хотел добавить ссылки к сообщению? Ссылки размешаются в кнопках к сообщению ")

	keyboard := botgolang.NewKeyboard()
	yes := botgolang.NewCallbackButton("Да", "/create_with_link")
	no := botgolang.NewCallbackButton("Нет", "/create_without_link")
	keyboard.AddRow(yes, no)
	message.AttachInlineKeyboard(keyboard)

	if err := message.Send(); err != nil {
		log.Printf("failed to send message: %s", err)
	}
}
func (us *UserState) SendRequireForLink(userID string) {
	message := vkbot.GetBot().NewTextMessage(userID, "Введи ссылку для добавления к апруву. Ссылка должна начинаться с http/https. "+
		"Если ты хочешь добавить несколько ссылок, то введи их через запятую. "+
		"\nВот пример: https://example.com, https://example.com/, https://example.com/somepath\n"+
		"Не больше 8 ссылок")

	if err := message.Send(); err != nil {
		log.Printf("failed to send message: %s", err)
	}
}
func SendNeedFile(userID string) {
	message := vkbot.GetBot().NewTextMessage(userID, "Нужно добавить файлы к апруву?")
	keyboard := botgolang.NewKeyboard()
	yes := botgolang.NewCallbackButton("Да", "/create_with_file")
	no := botgolang.NewCallbackButton("Нет", "/create_without_file")
	keyboard.AddRow(yes, no)
	message.AttachInlineKeyboard(keyboard)
	if err := message.Send(); err != nil {
		log.Printf("failed to send message: %s", err)
	}
}
func SendRequireParticipants(userID string) {
	message := vkbot.GetBot().NewMessage(userID)

	message.Text = "Почти что готово! Введи через запятую логины участников, которые должны подтвердить апрув. \n" +
		"Ты можешь быстро добавлять контакты из недавных чатов, введи @ чтобы отобразился список контактов"

	file, err := os.Open("./internal/assets/tooltip.png")
	if err != nil {
		log.Println("cannot open file: %s", err)
	}

	if file != nil {
		message.File = file
	}

	if err := message.Send(); err != nil {
		log.Printf("failed to send message: %s", err)
	}
	if err = file.Close(); err != nil {
		log.Println("failed to close file: %s", err)
	}
	defer file.Close()
}
func (us *UserState) SetLinks(userID string, links []string) {
	us.Links = links
	if us.EditMode {
		us.StopEditAndSendExample(userID)
		return
	}
	us.CurrentStage = NeedFile
	message := vkbot.GetBot().NewTextMessage(userID, "Успешно добавлено! Нужно добавить файлы к апруву?")
	keyboard := botgolang.NewKeyboard()
	yes := botgolang.NewCallbackButton("Да", "/create_with_file")
	no := botgolang.NewCallbackButton("Нет", "/create_without_file")
	keyboard.AddRow(yes, no)
	message.AttachInlineKeyboard(keyboard)
	if err := message.Send(); err != nil {
		log.Printf("failed to send message: %s", err)
	}

	return
}
func (us *UserState) ParseDuration(text string, userID string) bool {
	duration, err := utlis.DefineTime(text)
	if err != nil {
		var message = vkbot.GetBot().NewMessage(userID)
		message.Text = "Я не смог понять в течение какого времени должен быть подписан апрув."
		if errors.Is(botErrors.ErrHoursBelowMinimum, err) {
			message.Text = "Указываемое время не может быть меньше 1 часа"
		}
		if errors.Is(botErrors.ErrHoursExceedLimit, err) {
			message.Text = "Указываемое время не может быть больше 2 месяцев"
		}
		if errors.Is(botErrors.ErrDateInPast, err) {
			message.Text = "Указанная дата уже прошла"
		}

		message.Text += "\nПопробуй ввести снова"
		if err1 := message.Send(); err1 != nil {
			log.Println(err1)
		}
		return false
	}
	us.ConfirmTime = duration
	return true

}
func (us *UserState) SetFile(a *botgolang.EventPayload) bool {
	var fileID string
	var ok bool
	for _, pts := range a.Parts {
		if pts.Type == botgolang.FILE {
			fileID = pts.Payload.FileID
			us.FileID = pts.Payload.FileID
			ok = true
		}
	}
	if !ok {
		us.CurrentStage = NeedFile
		message := vkbot.GetBot().NewTextMessage(a.From.ID, "Ты отправил сообшение в котором нет файла, давай попробуем заново")
		if err := message.Send(); err != nil {
			log.Printf("failed to send message: %s", err)
		}
		return false
	}

	us.FileID = fileID
	return true
}
func (us *UserState) SetParticipants(text, userID string) {

	var result = utlis.ExtractEmails(text)

	if result.AllValid == true && len(result.ValidEmails) != 0 {
		registered, notRegistered, err := repositories.CheckEmailsInDB(result.ValidEmails)
		if err != nil {
			log.Println(err)
		}

		var message = vkbot.GetBot().NewMessage(userID)
		if len(registered) != 0 {
			registered := utlis.CreateUserLink(registered)
			message.Text = fmt.Sprintf("Выбраны участники: %s \n"+
				"Они получат твой апрув как только ты подтвердишь создание \n", registered)
		}

		if len(notRegistered) != 0 {
			us.NotRegistered = notRegistered
			message.Text += fmt.Sprintf("Участники с этими почтами еще ни разу мне не писали: %s, но они все равно получат твой апрув как только напишут мне\n",
				strings.Join(notRegistered, ", "))
		}

		message.Text += "Я все правильно понял?"
		keyboard := botgolang.NewKeyboard()
		yes := botgolang.NewCallbackButton("Да", "/create_participants_ok")
		no := botgolang.NewCallbackButton("Не совсем", "/create_participants_again")
		keyboard.AddRow(yes, no)
		message.AttachInlineKeyboard(keyboard)
		if err := message.Send(); err != nil {
			log.Printf("failed to send message: %s", err)
		}

		us.Participants = result.ValidEmails
		return
	}
	if len(result.InvalidElements) != 0 {
		message := vkbot.GetBot().NewTextMessage(userID, fmt.Sprintf("Я не понял следующие значения: %s \n"+
			"Попробуй написать все значения через запятую и убедись что эти значения действительно являются почтами", strings.Join(result.InvalidElements, ", ")))
		log.Println(result.InvalidElements)
		if err := message.Send(); err != nil {
			log.Printf("failed to send message: %s", err)
		}
		return
	}
	var message = vkbot.GetBot().NewTextMessage(userID, "Я не смог понять ни одного логина. Давай попробуем заново")
	if err := message.Send(); err != nil {
		log.Println(err)
	}
	SendRequireParticipants(userID)

}

func (us *UserState) SendExampleMessage(userID string) {
	message := vkbot.GetBot().NewTextMessage(userID, "Твой апрув будет иметь следующий вид:")

	if err := message.Send(); err != nil {
		log.Printf("failed to send message: %s", err)
	}
	example := us.GetApprovalExample(userID)
	if err := example.Send(); err != nil {
		log.Println(err)
	}
}
func (us *UserState) GetApprovalExample(userId string) *botgolang.Message {
	exampleMessage := vkbot.GetBot().NewMessage(userId)
	if us.FileID != "" {
		exampleMessage.FileID = us.FileID
	}

	exampleMessage.Text = us.generateApproveText()
	keyboard := botgolang.NewKeyboard()
	if len(us.Links) != 0 {
		var links []botgolang.Button
		for id, el := range us.Links {
			link := botgolang.NewURLButton(fmt.Sprintf("#%v %s ", id, "cсылка"), el)
			links = append(links, link)
		}
		keyboard.AddRow(links...)
	}

	yesButton := botgolang.NewCallbackButton("✅Подтвердить", "/")
	if us.Cancelable {
		noButton := botgolang.NewCallbackButton("⛔️Отклонить", "/")
		keyboard.AddRow(yesButton, noButton)
	} else {
		keyboard.AddRow(yesButton)
	}
	exampleMessage.AttachInlineKeyboard(keyboard)
	return exampleMessage
	/*if err := exampleMessage.Send(); err != nil {
		log.Printf("failed to send message: %s", err)
	}*/
}

func SelectFileType(userID string) {
	message := vkbot.GetBot().NewTextMessage(userID, "Какой тип файла ты хочешь прикрепить? Редактируемый или обычный? \n"+
		"Редактируемый файл будет полезен в случае, если вам нужно собрать подписи на документ. Каждый из участников должен будет отправить обновлённую версию документа, чтобы подтвердить его \n"+
		"Обычный файл просто будет доступен для просмотра")
	keyboard := botgolang.NewKeyboard()
	editable := botgolang.NewCallbackButton("Редактируемый", "/create_editable_file")
	standard := botgolang.NewCallbackButton("Обычный", "/create_standard_file")
	keyboard.AddRow(editable, standard)
	message.AttachInlineKeyboard(keyboard)

	if err := message.Send(); err != nil {
		log.Printf("failed to send message: %s", err)
	}
}

func CheckLinks(text string, userID string) ([]string, bool) {
	links, ok := utlis.DefineLink(text)

	if len(links) > 8 {
		message := vkbot.GetBot().NewTextMessage(userID, "Не больше 8 ссылок. Попробуй снова?")
		if err := message.Send(); err != nil {
			log.Printf("failed to send message: %s", err)
		}
		return links, false
	}

	if !ok {
		message := vkbot.GetBot().NewTextMessage(userID, fmt.Sprintf("Я не смог понять следующую ссылку(-и): %s \n"+
			"Давай еще раз: Введи ссылку для добавления к апруву. Ссылка должна начинаться с http/https."+
			"Если ты хочешь добавить несколько ссылок, то введи их через запятую."+
			"\nВот пример: https://example.com, https://example.com/, https://example.com/somepath ", strings.Join(links, ", ")))

		if err := message.Send(); err != nil {
			log.Printf("failed to send message: %s", err)
		}
	}
	return links, ok
}
func AskFile(userID string) {
	message := vkbot.GetBot().NewTextMessage(userID, "Отправь файл, который необходимо прикрепить к апруву")
	if err := message.Send(); err != nil {
		log.Printf("failed to send message: %s", err)
	}
}
func sendFinalOkMessage(userID string) {
	message := vkbot.GetBot().NewTextMessage(userID, "Отправить апрув всем участникам или нужно что-то изменить?")

	keyboard := botgolang.NewKeyboard()
	yes := botgolang.NewCallbackButton("Отправить", "/create_yes_send")
	no := botgolang.NewCallbackButton("Изменить", "/create_not_send")
	keyboard.AddRow(yes, no)
	message.AttachInlineKeyboard(keyboard)

	if err := message.Send(); err != nil {
		log.Printf("failed to send message: %s", err)
	}
}

func SendChangeMessage(userID string) {
	message := vkbot.GetBot().NewTextMessage(userID, "Ты можешь обновить любую часть апрува. Выбери, что именно хочешь изменить, нажав одну из кнопок ниже:")

	keyboard := botgolang.NewKeyboard()
	title := botgolang.NewCallbackButton("Заголовок", "/change_title")
	description := botgolang.NewCallbackButton("Описание", "/change_description")
	visible := botgolang.NewCallbackButton("Видимость", "/change_visible")
	cancel := botgolang.NewCallbackButton("Возможность отмены", "/change_cancel")
	duration := botgolang.NewCallbackButton("Время", "/change_duration")
	link := botgolang.NewCallbackButton("Ссылки", "/change_link")
	participants := botgolang.NewCallbackButton("Участники", "/change_participants")
	file := botgolang.NewCallbackButton("Файл", "/change_file")
	not := botgolang.NewCallbackButton("Ничего не менять", "/not_change").WithStyle(botgolang.ButtonAttention)
	keyboard.AddRow(title, description, visible)
	keyboard.AddRow(cancel, duration)
	keyboard.AddRow(link, participants, file)
	keyboard.AddRow(not)
	message.AttachInlineKeyboard(keyboard)
	if err := message.Send(); err != nil {
		log.Printf("failed to send message: %s", err)
	}

}

func SendApproveCreatedMessage(userID string) {
	text := "Заявка успешно создана!\nОтправь мне /manage для того чтобы управлять созданными апрувами"
	notRegistered := UserStates[userID].NotRegistered
	if len(notRegistered) != 0 {
		text += fmt.Sprintf("\nНе забудь напомнить следующим пользователям написать мне: %s, иначе я не смогу отправить им твой апрув",
			strings.Join(notRegistered, ", "))
	}
	message := vkbot.GetBot().NewTextMessage(userID, text)
	if err := message.Send(); err != nil {
		log.Printf("failed to send message: %s", err)
	}
	delete(UserStates, userID)
}

func (us *UserState) StopEditAndSendExample(userID string) {
	us.CurrentStage = ExampleMessage
	us.SendExampleMessage(userID)
	sendFinalOkMessage(userID)
	us.EditMode = false
}

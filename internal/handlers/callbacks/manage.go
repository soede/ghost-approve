package callbacks

import (
	"errors"
	"fmt"
	"ghost-approve/internal/notifier"
	"ghost-approve/internal/repositories"
	"ghost-approve/internal/services"
	"ghost-approve/pkg/vkbot"
	botgolang "github.com/mail-ru-im/bot-golang"
	"gorm.io/gorm"
	"log"
	"strconv"
	"strings"
	"sync"
)

func handleManageCommand(data string, userID string) {
	command := strings.TrimPrefix(data, "/manage_")
	if command == "approves" {
		services.SendManageApprovals(userID)
		return
	}
	if command == "reports" {
		var err error
		messages, err := services.ReportsApprovalsByUserID(userID)
		if err != nil {
			return
		}

		//отправляет все отчеты
		var wg sync.WaitGroup
		for _, message := range messages {
			messageCopy := message
			wg.Add(1)
			go func(msg *botgolang.Message) {
				defer wg.Done()
				err := msg.Send()
				if err != nil {
					log.Println(err)
				}
			}(messageCopy)
		}
		wg.Wait()

		return
	}

	//функции с параметром
	if strings.HasPrefix(command, "notification_") {
		approveID, err := strconv.Atoi(strings.TrimPrefix(command, "notification_"))
		if err != nil {
			log.Println(err)
			return
		}
		isCurrent, err := services.IsCurrentApprove(approveID)
		if !isCurrent {
			err = vkbot.GetBot().NewTextMessage(userID, fmt.Sprintf("Апрув #%d уже не актуален", approveID)).Send()
			if err != nil {
				log.Println(err)
			}
			return
		}
		users, err := repositories.ApprovalParticipants(approveID)
		if err != nil {
			log.Println(err)
			return
		}

		err = notifier.CustomRemind(userID, users, approveID)
		if err != nil {
			log.Println(err)
		}

	}
	if strings.HasPrefix(command, "report_") {
		approveID, err := strconv.Atoi(strings.TrimPrefix(command, "report_"))
		if err != nil {
			log.Println(err)
			return
		}
		message, err := services.ReportMessage(approveID, userID)
		if message == nil || err != nil {
			err = vkbot.GetBot().NewTextMessage(userID, fmt.Sprintf("Апрув #%d не актуален", approveID)).Send()
			if err != nil {
				log.Println(err)
			}
			return
		}
		err = message.Send()
		if err != nil {
			log.Println(err)
		}

	}
	if strings.HasPrefix(command, "statistic_") {
		approveID, err := strconv.Atoi(strings.TrimPrefix(command, "statistic_"))
		if access, err := services.CheckApprovalAccess(approveID, userID); !access || err != nil {
			vkbot.GetBot().NewTextMessage(userID, "У тебя нет доступа к этому апруву").Send()
			return
		}
		if err != nil {
			log.Println(err)
			return
		}
		message := vkbot.GetBot().NewMessage(userID)
		message.Text += fmt.Sprintf("📊Cтатистика апрува #%d\n", approveID)
		message.Text += services.FetchStats(approveID)
		err = message.Send()
		if err != nil {
			log.Println(err)
		}
	}
	if strings.HasPrefix(command, "events_") {
		approveID, err := strconv.Atoi(strings.TrimPrefix(command, "events_"))
		if err != nil {
			log.Println(err)
			return
		}
		if access, err := services.CheckApprovalAccess(approveID, userID); !access || err != nil {
			vkbot.GetBot().NewTextMessage(userID, "У тебя нет доступа к этому апруву").Send()
			return
		}
		message := vkbot.GetBot().NewMessage(userID)
		text, err := services.FetchEvents(approveID)
		if errors.Is(gorm.ErrRecordNotFound, err) {
			vkbot.GetBot().NewTextMessage(userID, fmt.Sprintf("Апрув #%d не актуален", approveID))
		}
		message.Text += fmt.Sprintf("🕗История действий апрува #%d\n", approveID)
		message.Text += text
		err = message.Send()
		if err != nil {
			log.Println(err)
		}
	}
	if strings.HasPrefix(command, "hide_") {
		approveID, err := strconv.Atoi(strings.TrimPrefix(command, "hide_"))
		if err != nil {
			log.Println(err)
			return
		}
		err = repositories.HideReportForUser(userID, approveID)
		if err != nil {
			log.Println(err)
		}
		err = vkbot.GetBot().NewTextMessage(userID, fmt.Sprintf("Апрув #%d больше не будет отображаться в отчетах. Автор апрува все еще имеет доступ к отчету.", approveID)).Send()
		if err != nil {
			log.Println(err)
		}
	}
	if strings.HasPrefix(command, "cancel_") {
		approveID, err := strconv.Atoi(strings.TrimPrefix(command, "cancel_"))
		if err != nil {
			log.Println(err)
			return
		}

		isCurrent, err := services.IsCurrentApprove(approveID)
		if !isCurrent {
			err := vkbot.GetBot().NewTextMessage(userID, fmt.Sprintf("Апрув #%d уже не актуален", approveID)).Send()
			if err != nil {
				log.Println(err)
			}
			return
		}

		message := vkbot.GetBot().NewMessage(userID)
		text := fmt.Sprintf("Ты действительно хочешь отменить апрув #%d? Все участники получат уведомление о том, что апрув был отменен тобой", approveID)
		keyboard := botgolang.NewKeyboard()
		yes := botgolang.NewCallbackButton("Да", fmt.Sprintf("/manage_delete_%d", approveID)).WithStyle(botgolang.ButtonAttention)
		no := botgolang.NewCallbackButton("Нет", fmt.Sprintf("/manage_notDelete_%d", approveID))

		keyboard.AddRow(yes, no)
		message.AttachInlineKeyboard(keyboard)
		message.Text = text
		err = message.Send()
		if err != nil {
			log.Println(err)
		}
	}
	if strings.HasPrefix(command, "delete_") {
		approveID, err := strconv.Atoi(strings.TrimPrefix(command, "delete_"))

		if err != nil {
			log.Println(err)
			return
		}

		isCurrent, err := services.IsCurrentApprove(approveID)
		if !isCurrent {
			err := vkbot.GetBot().NewTextMessage(userID, fmt.Sprintf("Апрув #%d уже не актуален", approveID)).Send()
			if err != nil {
				log.Println(err)
			}
			return
		}
		err = services.CancelApprove(approveID)
		if err != nil {
			log.Println(err)
			return
		}

		err = services.CancelMessageToUsers(approveID, userID)
		if err != nil {
			log.Println(err)
		}

		err = vkbot.GetBot().NewTextMessage(userID, fmt.Sprintf("Апрув #%d был отменён.\nИнформацию о нем можно найти если ввести /report%d", approveID, approveID)).Send()
		if err != nil {
			log.Println(err)
		}
	}
	if strings.HasPrefix(command, "ask_dreport") {
		approveID, err := strconv.Atoi(strings.TrimPrefix(command, "ask_dreport_"))
		if err != nil {
			log.Println(err)
			return
		}

		_, err = repositories.GetApprovalByID(approveID)
		if err != nil {
			err = vkbot.GetBot().NewTextMessage(userID, fmt.Sprintf("Апрув #%d уже не актуален", approveID)).Send()
			if err != nil {
				log.Println(err)
			}
			return
		}

		message := vkbot.GetBot().NewMessage(userID)
		message.Text = fmt.Sprintf("Ты действительно хочешь удалить апрув #%d?", approveID)
		keyboard := botgolang.NewKeyboard()
		yes := botgolang.NewCallbackButton("Да", fmt.Sprintf("/manage_dreport_%d", approveID)).WithStyle(botgolang.ButtonAttention)
		no := botgolang.NewCallbackButton("Нет", fmt.Sprintf("/manage_notDelete_%d", approveID))
		keyboard.AddRow(yes, no)
		message.AttachInlineKeyboard(keyboard)
		err = message.Send()
		if err != nil {
			log.Println(err)
		}
	}
	if strings.HasPrefix(command, "dreport_") {
		approveID, err := strconv.Atoi(strings.TrimPrefix(command, "dreport_"))
		if err != nil {
			log.Println(err)
			return
		}
		_, err = repositories.GetApprovalByID(approveID)
		if err != nil {
			err = vkbot.GetBot().NewTextMessage(userID, fmt.Sprintf("Апрув #%d уже не актуален", approveID)).Send()
			if err != nil {
				log.Println(err)
			}
			return
		}
		err = repositories.DeleteApprovalByID(approveID)
		if err != nil {
			err = vkbot.GetBot().NewTextMessage(userID, fmt.Sprintf("Не удалось удалить апрув #%d", approveID)).Send()
		}
		err = vkbot.GetBot().NewTextMessage(userID, fmt.Sprintf(" Апрув #%d был удален", approveID)).Send()

	}
	if strings.HasPrefix(command, "notDelete_") {
		approveID, err := strconv.Atoi(strings.TrimPrefix(command, "notDelete_"))
		if err != nil {
			log.Println(err)
			return
		}

		if vkbot.GetBot().NewTextMessage(userID, fmt.Sprintf("Апрув #%d не был удален", approveID)).Send(); err != nil {
			log.Println(err)
		}
		return
	}
}

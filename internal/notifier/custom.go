package notifier

import (
	"fmt"
	"ghost-approve/internal/repositories"
	"ghost-approve/internal/utils"
	"ghost-approve/pkg/vkbot"
	log "github.com/sirupsen/logrus"
	"sync"
)

func SelectNotificationUsers(userID string, approveID int) {
	message := vkbot.GetBot().NewTextMessage(userID, "Все участники, которые не откликнулись на апрув, получили напоминание")
	err := message.Send()
	if err != nil {
		log.Error(err)
	}
}

func CustomRemind(authorID string, users []string, approveID int) error {
	expired, err := repositories.IsReminderExpired(approveID)
	if err != nil {
		return err
	}
	if !expired {
		err = vkbot.GetBot().NewTextMessage(authorID, "Нельзя отправлять напоминания чаще 1 раза в час, попробуй позже").Send()
		if err != nil {
			return err
		}
		return nil
	}

	var wg sync.WaitGroup
	link := utils.CreateUserLink([]string{authorID})
	if err != nil {
		return err
	}

	text := fmt.Sprintf("%s напоминает о том, что нужно подписать апрув #%d", link, approveID)

	err = repositories.CreateRemind(approveID)
	for _, userID := range users {
		userID := userID
		if userID == authorID {
			continue
		}
		go func() {
			wg.Add(1)
			err = vkbot.GetBot().NewTextMessage(userID, text).Send()
			if err != nil {
				log.Error(err)
			}
			defer wg.Done()
		}()
	}
	wg.Wait()

	SelectNotificationUsers(authorID, approveID)
	return err

}

// Отправляет всем заругестрированным участникам апрува сообщение
func NotifyAll(approvalID int, authorID, text string) error {
	usersIDs, err := repositories.RegisteredUsersID(approvalID)
	if err != nil {
		return err
	}

	var wg sync.WaitGroup
	for _, p := range usersIDs {
		if p == authorID {
			continue
		}
		message := vkbot.GetBot().NewMessage(p)

		message.Text = text

		go func() {
			wg.Add(1)
			if err := message.Send(); err != nil {
				log.Errorf("failed to send message: %s", err)
			}
			defer wg.Done()
		}()
	}
	wg.Wait()
	return err

}

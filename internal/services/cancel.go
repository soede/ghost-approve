package services

import (
	"fmt"
	"ghost-approve/internal/repositories"
	"ghost-approve/pkg/vkbot"
	"log"
	"sync"
)

func CancelMessageToUsers(approveID int, authorID string) error {
	usersIDs, err := repositories.RegisteredUsersID(approveID)
	if err != nil {
		return err
	}

	var wg sync.WaitGroup
	for _, p := range usersIDs {
		if p == authorID {
			continue
		}
		message := vkbot.GetBot().NewMessage(p)

		message.Text = fmt.Sprintf("Апрув #%d был отменен автором. Информацию о нем можно получить с помощью команды /received ", approveID)

		go func() {
			wg.Add(1)
			if err := message.Send(); err != nil {
				log.Printf("failed to send message: %s", err)
			}
			defer wg.Done()
		}()
	}
	wg.Wait()
	return err
}


package callbacks

import (
	"errors"
	"ghost-approve/internal/services"
	"ghost-approve/pkg/botErrors"
	"ghost-approve/pkg/vkbot"
	botgolang "github.com/mail-ru-im/bot-golang"
	log "github.com/sirupsen/logrus"
	"strconv"
	"strings"
)

// handleRejectCallback обрабатывает reject_* команды
func handleRejectCallback(p *botgolang.EventPayload, data string) {
	approveIDStr := strings.TrimPrefix(data, "/reject_")
	approveID, err := strconv.Atoi(approveIDStr)
	if err != nil {
		log.Errorf("Ошибка преобразования ID: %s в int: %v", approveIDStr, err)
	}

	err = services.RejectApprove(approveID, p.From.ID)

	if errors.Is(err, botErrors.ErrAlreadyHasResponse) {
		err = vkbot.GetBot().NewTextMessage(p.From.ID, "Ты уже дал отклик по этому апруву").Send()
	}
	if errors.Is(err, botErrors.ErrApprovalIsNotRelevant) {
		err = vkbot.GetBot().NewTextMessage(p.From.ID, "Этот апрув уже не актуален").Send()
	}
	if errors.Is(err, botErrors.ErrNoAccess) {
		err = vkbot.GetBot().NewTextMessage(p.From.ID, "Нет доступа к апруву").Send()
	}

	if err != nil {
		log.Errorf("Ошибка подтверждения: %v", err)
	}
}

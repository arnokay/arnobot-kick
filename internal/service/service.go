package service

import (
	"github.com/arnokay/arnobot-shared/service"
)

type Services struct {
	AuthModule         *service.AuthModule
	PlatformModule     *service.PlatformModuleOut
	KickManager        *KickManager
	BotService         *BotService
	WebhookService     *WebhookService
	KickService      *KickService
	TransactionService service.ITransactionService
}

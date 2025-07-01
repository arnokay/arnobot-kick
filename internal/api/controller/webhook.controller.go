package controller

import (
	"log/slog"
	"strconv"

	"github.com/arnokay/arnobot-shared/applog"
	"github.com/arnokay/arnobot-shared/events"
	"github.com/arnokay/arnobot-shared/platform"
	sharedService "github.com/arnokay/arnobot-shared/service"
	"github.com/labstack/echo/v4"
	"github.com/scorfly/gokick"

	"github.com/arnokay/arnobot-kick/internal/api/middleware"
	"github.com/arnokay/arnobot-kick/internal/data"
	"github.com/arnokay/arnobot-kick/internal/service"
)

type WebhookController struct {
	logger *slog.Logger

	middlewares *middleware.Middlewares

	kickService    *service.KickService
	botService     *service.BotService
	platformModule *sharedService.PlatformModuleOut
}

func NewWebhookController(
	middlewares *middleware.Middlewares,
	botService *service.BotService,
	platformModule *sharedService.PlatformModuleOut,
) *WebhookController {
	logger := applog.NewServiceLogger("ChatController")

	return &WebhookController{
		logger: logger,

		middlewares:    middlewares,
		botService:     botService,
		platformModule: platformModule,
	}
}

func (c *WebhookController) Routes(parentGroup *echo.Group) {
	parentGroup.POST("/callback", c.Callback, c.middlewares.VerifyKickWebhook)
}

func (c *WebhookController) Callback(ctx echo.Context) error {
	switch ctx.Request().Header.Get("Kick-Event-Type") {
	case gokick.SubscriptionNameChatMessage.String():
		var event gokick.ChatMessageEvent
		ctx.Bind(&event)

		bot, err := c.botService.SelectedBotGetByBroadcasterID(ctx.Request().Context(), int(event.Broadcaster.UserID))
		if err != nil {
			c.logger.ErrorContext(ctx.Request().Context(), "cannot get selected bot")
			return nil
		}

		broadcasterID := strconv.Itoa(event.Broadcaster.UserID)
		chatterID := strconv.Itoa(event.Sender.UserID)

		internalEvent := events.Message{
			EventCommon: events.EventCommon{
				Platform:      platform.Kick,
				BroadcasterID: broadcasterID,
				UserID:        bot.UserID,
				BotID:         strconv.Itoa(int(bot.BotID)),
			},
			MessageID:        event.MessageID,
			Message:          event.Content,
			ReplyTo:          "",
			BroadcasterLogin: event.Broadcaster.Username,
			BroadcasterName:  event.Broadcaster.Username,
			ChatterID:        chatterID,
			ChatterName:      event.Sender.Username,
			ChatterRole:      data.GetChatterRole(event.Sender.Identity.Badges),
			ChatterLogin:     event.Sender.Username,
		}

		err = c.platformModule.ChatMessageNotify(ctx.Request().Context(), internalEvent)
		if err != nil {
			c.logger.ErrorContext(ctx.Request().Context(), "cannot notify message")
			return nil
		}
	}

	return nil
}

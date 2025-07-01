package controller

import (
	"fmt"
	"log/slog"
	"strconv"

	"github.com/arnokay/arnobot-shared/applog"
	"github.com/arnokay/arnobot-shared/apptype"
	"github.com/arnokay/arnobot-shared/data"
	"github.com/arnokay/arnobot-shared/events"
	"github.com/arnokay/arnobot-shared/pkg/assert"
	"github.com/arnokay/arnobot-shared/platform"
	sharedService "github.com/arnokay/arnobot-shared/service"
	"github.com/arnokay/arnobot-shared/topics"
	"github.com/nats-io/nats.go"

	"github.com/arnokay/arnobot-kick/internal/service"
)

type ChatController struct {
	kickService *service.KickService
	authModule  *sharedService.AuthModule

	logger *slog.Logger
}

func NewChatController(
	kickService *service.KickService,
	authModule *sharedService.AuthModule,
) *ChatController {
	logger := applog.NewServiceLogger("mb-chat-controller")

	return &ChatController{
		kickService: kickService,
		authModule:  authModule,

		logger: logger,
	}
}

func (c *ChatController) Connect(conn *nats.Conn) {
	topic := topics.
		TopicBuilder(topics.PlatformBroadcasterChatMessageSend).
		Platform(platform.Kick).
		BroadcasterID(topics.Any).
		Build()
	_, err := conn.QueueSubscribe(
		topic,
		topic,
		c.ChatMessageSend,
	)
	assert.NoError(err, fmt.Sprintf("MBChatController cannot subscribe to the topic: %s", topic))
}

func (c *ChatController) ChatMessageSend(msg *nats.Msg) {
	var payload apptype.Request[events.MessageSend]

	payload.Decode(msg.Data)

	ctx, cancel := newControllerContext(payload.TraceID)
	defer cancel()

	botProvider, err := c.authModule.AuthProviderGet(ctx, data.AuthProviderGet{
		ProviderUserID: &payload.Data.BotID,
		Provider:       platform.Kick.String(),
	})
	if err != nil {
		c.logger.ErrorContext(ctx, "cant access auth module")
		return
	}

	broadcasterID, err := strconv.Atoi(payload.Data.BroadcasterID)
	if err != nil {
		c.logger.ErrorContext(ctx, "cant parse to string", "err", err, "broadcasterID", payload.Data.BroadcasterID)
	}
	err = c.kickService.AppSendChannelMessage(ctx, *botProvider, broadcasterID, payload.Data.Message, payload.Data.ReplyTo)
	if err != nil {
		c.logger.ErrorContext(
			ctx,
			"cannot send message to channel",
			"payload", payload,
		)
		return
	}
}

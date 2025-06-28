package service

import (
	"context"
	"log/slog"

	"github.com/arnokay/arnobot-shared/apperror"
	"github.com/arnokay/arnobot-shared/applog"
	"github.com/arnokay/arnobot-shared/data"
	"github.com/scorfly/gokick"
)

type KickService struct {
	kickManager *KickManager
	logger      *slog.Logger
}

func NewKickService(
	kickManager *KickManager,
) *KickService {
	logger := applog.NewServiceLogger("kick-service")

	return &KickService{
		kickManager: kickManager,
		logger:      logger,
	}
}

func (s *KickService) AppSendChannelMessage(
	ctx context.Context,
	botProvider data.AuthProvider,
	broadcasterID int,
	message string,
	replyTo string,
) error {
	client := s.kickManager.GetByProvider(ctx, botProvider)

	_, err := client.SendChatMessage(ctx, int(broadcasterID), message, &replyTo, gokick.MessageTypeUser)
	if err != nil {
		s.logger.ErrorContext(
			ctx,
			"cannot send message to chat",
			"err", err,
			"broadcasterID", broadcasterID,
			"botID", botProvider.ProviderUserID,
			"message", message,
			"replyTo", replyTo,
		)
		return apperror.ErrExternal
	}

	return nil
}

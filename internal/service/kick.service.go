package service

import (
	"context"
	"strconv"

	"github.com/arnokay/arnobot-shared/apperror"
	"github.com/arnokay/arnobot-shared/applog"
	"github.com/arnokay/arnobot-shared/data"
	"github.com/scorfly/gokick"
)

type KickService struct {
	kickManager *KickManager
	logger      applog.Logger
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
	broadcasterID string,
	message string,
	replyTo string,
) error {
	client := s.kickManager.GetByProvider(ctx, botProvider)
	bID, err := strconv.Atoi(broadcasterID)
	if err != nil {
		s.logger.ErrorContext(ctx, "cannot convert broadcasterID to int", "broadcaster_id", broadcasterID)
		return apperror.ErrInvalidInput
	}

	_, err = client.SendChatMessage(ctx, bID, message, &replyTo, gokick.MessageTypeUser)
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

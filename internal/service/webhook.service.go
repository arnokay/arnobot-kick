package service

import (
	"context"
	"strconv"

	"github.com/arnokay/arnobot-shared/apperror"
	"github.com/arnokay/arnobot-shared/applog"
	"github.com/arnokay/arnobot-shared/data"
	"github.com/scorfly/gokick"

	"github.com/arnokay/arnobot-kick/internal/config"
)

type WebhookService struct {
	kickManager *KickManager
	kickService *KickService

	logger applog.Logger

	callbackURL string
}

func NewWebhookService(
	helixManager *KickManager,
	kickService *KickService,
) *WebhookService {
	logger := applog.NewServiceLogger("webhook-service")

	return &WebhookService{
		kickManager: helixManager,
		kickService: kickService,
		logger:      logger,
		callbackURL: config.Config.Webhooks.Callback,
	}
}

func (s *WebhookService) UnsubscribeMany(
	ctx context.Context,
	botProvider data.AuthProvider,
	subscriptionIds []string,
) error {
	client := s.kickManager.GetByProvider(ctx, botProvider)

	_, err := client.DeleteSubscriptions(ctx, gokick.NewSubscriptionToDeleteFilter().SetIDs(subscriptionIds))
	if err != nil {
		s.logger.ErrorContext(
			ctx,
			"cannot unsubscribe",
			"err", err,
		)
		return apperror.ErrExternal
	}

	return nil
}

func (s *WebhookService) Unsubscribe(
	ctx context.Context,
	botProvider data.AuthProvider,
	subscriptionID string,
) error {
	return s.UnsubscribeMany(ctx, botProvider, []string{subscriptionID})
}

func (s *WebhookService) UnsubscribeAll(
	ctx context.Context,
	botProvider data.AuthProvider,
	broadcasterID string,
) error {
	client := s.kickManager.GetByProvider(ctx, botProvider)

	var subIds []string

	subs, err := client.GetSubscriptions(ctx)
	if err != nil {
		s.logger.ErrorContext(ctx, "error getting eventsub subscriptions", "err", err)
		return apperror.ErrExternal
	}

  bID, err := strconv.Atoi(broadcasterID)
  if err != nil {
    s.logger.ErrorContext(ctx, "cannot convert broadcasterID", "broadcaster_id", broadcasterID)
    return apperror.ErrInvalidInput
  }
	for _, sub := range subs.Result {
		if sub.BroadcasterUserID == bID {
			subIds = append(subIds, sub.ID)
		}
	}

	err = s.UnsubscribeMany(ctx, botProvider, subIds)
	if err != nil {
		s.logger.ErrorContext(ctx, "failed to unsubscribe", "err", err)
		return err
	}

	return nil
}

func (s *WebhookService) Subscribe(
	ctx context.Context,
	broadcasterProvider data.AuthProvider,
) error {
	client := s.kickManager.GetByProvider(ctx, broadcasterProvider)

	_, err := client.CreateSubscriptions(
		ctx,
		gokick.SubscriptionMethodWebhook,
		[]gokick.SubscriptionRequest{
			{Name: gokick.SubscriptionNameChatMessage, Version: 1},
			{Name: gokick.SubscriptionNameChannelFollow, Version: 1},
			{Name: gokick.SubscriptionNameChannelSubscriptionRenewal, Version: 1},
			{Name: gokick.SubscriptionNameChannelSubscriptionGifts, Version: 1},
			{Name: gokick.SubscriptionNameChannelSubscriptionCreated, Version: 1},
			{Name: gokick.SubscriptionNameLivestreamStatusUpdated, Version: 1},
			{Name: gokick.SubscriptionNameLivestreamMetadataUpdated, Version: 1},
			{Name: gokick.SubscriptionNameModerationBanned, Version: 1},
		},
		nil,
	)
	if err != nil {
		s.logger.ErrorContext(ctx, "cannot subscribe to channel", "err", err, "broadcasterID", broadcasterProvider.ProviderUserID)
		return apperror.ErrExternal
	}

	return nil
}

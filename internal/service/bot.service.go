package service

import (
	"context"

	"github.com/arnokay/arnobot-shared/apperror"
	"github.com/arnokay/arnobot-shared/applog"
	"github.com/arnokay/arnobot-shared/data"
	"github.com/arnokay/arnobot-shared/db"
	"github.com/arnokay/arnobot-shared/platform"
	sharedService "github.com/arnokay/arnobot-shared/service"
	"github.com/arnokay/arnobot-shared/storage"
	"github.com/google/uuid"
)

type BotService struct {
	storage     storage.Storager
	txService   sharedService.ITransactionService
	authModule  *sharedService.AuthModule
	whService   *WebhookService
	kickService *KickService

	logger applog.Logger
}

func NewBotService(
	store storage.Storager,
	txService sharedService.ITransactionService,
	authModule *sharedService.AuthModule,
	whService *WebhookService,
	kickService *KickService,
) *BotService {
	logger := applog.NewServiceLogger("bot-service")
	return &BotService{
		storage:     store,
		txService:   txService,
		authModule:  authModule,
		whService:   whService,
		kickService: kickService,
		logger:      logger,
	}
}

func (s *BotService) StartBot(ctx context.Context, arg data.PlatformBotToggle) error {
	txCtx, err := s.txService.Begin(ctx)
	defer s.txService.Rollback(txCtx)
	if err != nil {
		return err
	}

	selectedBot, err := s.SelectedBotGet(txCtx, arg.UserID)
	if err != nil {
		if !apperror.IsAppErr(err) {
			return err
		}

		selectedBot, err = s.SelectedBotSetDefault(txCtx, arg.UserID)
		if err != nil {
			return err
		}
	}

	err = s.txService.Commit(txCtx)
	if err != nil {
		return err
	}

	broadcasterProvider, err := s.authModule.AuthProviderGet(ctx, data.AuthProviderGet{
		ProviderUserID: &selectedBot.BroadcasterID,
		Provider:       platform.Kick.String(),
	})
	if err != nil {
		s.logger.ErrorContext(ctx, "cannot get broadcaster provider")
		return err
	}

	err = s.whService.Subscribe(ctx, *broadcasterProvider)
	if err != nil {
		return err
	}

	err = s.SelectedBotChangeStatus(ctx, arg.UserID, true)
	if err != nil {
		return err
	}

	botProvider, err := s.authModule.AuthProviderGet(ctx, data.AuthProviderGet{
		ProviderUserID: &selectedBot.BotID,
		Provider:       platform.Kick.String(),
	})
	if err != nil {
		return err
	}

	s.kickService.AppSendChannelMessage(ctx, *botProvider, selectedBot.BroadcasterID, "hi!", "")

	return nil
}

func (s *BotService) StopBot(ctx context.Context, arg data.PlatformBotToggle) error {
	selectedBot, err := s.SelectedBotGet(ctx, arg.UserID)
	if err != nil {
		return err
	}

	botProvider, err := s.authModule.AuthProviderGet(ctx, data.AuthProviderGet{
		ProviderUserID: &selectedBot.BotID,
		Provider:       platform.Kick.String(),
	})
	if err != nil {
		s.logger.ErrorContext(ctx, "cannot get bot provider")
		return err
	}
	err = s.whService.UnsubscribeAll(ctx, *botProvider, selectedBot.BroadcasterID)
	if err != nil {
		s.logger.DebugContext(ctx, "bot cannot unsubscribe")
		return err
	}

	err = s.SelectedBotChangeStatus(ctx, arg.UserID, false)
	if err != nil {
		return err
	}

	return nil
}

func (s *BotService) SelectedBotSetDefault(ctx context.Context, userID uuid.UUID) (data.PlatformSelectedBot, error) {
	var bot data.PlatformBot

	txCtx, err := s.txService.Begin(ctx)
	defer s.txService.Rollback(txCtx)
	if err != nil {
		return data.PlatformSelectedBot{}, err
	}

	bots, err := s.BotsGet(ctx, data.PlatformBotsGet{
		UserID: &userID,
	})
	if err != nil {
		return data.PlatformSelectedBot{}, err
	}

	if len(bots) != 0 {
		bot = bots[0]
	} else {
		defaultBot, err := s.DefaultBotGet(ctx)
		if err != nil {
			return data.PlatformSelectedBot{}, err
		}
		userProvider, err := s.authModule.AuthProviderGet(ctx, data.AuthProviderGet{
			UserID:   &userID,
			Provider: platform.Kick.String(),
		})
		if err != nil {
			return data.PlatformSelectedBot{}, err
		}
		bot, err = s.BotCreate(ctx, data.PlatformBotCreate{
			UserID:        userID,
			BotID:         defaultBot.BotID,
			BroadcasterID: userProvider.ProviderUserID,
		})
		if err != nil {
			return data.PlatformSelectedBot{}, err
		}
	}

	selectedBot, err := s.SelectedBotChange(ctx, bot)
	if err != nil {
		return data.PlatformSelectedBot{}, err
	}

	err = s.txService.Commit(txCtx)
	if err != nil {
		return data.PlatformSelectedBot{}, err
	}

	return selectedBot, nil
}

func (s *BotService) SelectedBotGetByBroadcasterID(ctx context.Context, broadcasterID string) (data.PlatformSelectedBot, error) {
	fromDB, err := s.storage.Query(ctx).KickSelectedBotGetByBroadcasterID(ctx, broadcasterID)
	if err != nil {
		s.logger.DebugContext(ctx, "cannot get selected bot", "err", err, "broadcasterID", broadcasterID)
		return data.PlatformSelectedBot{}, s.storage.HandleErr(ctx, err)
	}

	bot := data.PlatformSelectedBot{
		UserID:        fromDB.UserID,
		BotID:         fromDB.BotID,
		BroadcasterID: fromDB.BroadcasterID,
		Enabled:       fromDB.Enabled,
		UpdatedAt:     fromDB.UpdatedAt,
	}

	return bot, nil
}

func (s *BotService) BotCreate(ctx context.Context, arg data.PlatformBotCreate) (data.PlatformBot, error) {
	fromDB, err := s.storage.Query(ctx).KickBotCreate(ctx, db.KickBotCreateParams{
		UserID:        arg.UserID,
		BroadcasterID: arg.BroadcasterID,
		BotID:         arg.BotID,
	})
	if err != nil {
		s.logger.DebugContext(ctx, "cannot create bot", "err", err)
		return data.PlatformBot{}, s.storage.HandleErr(ctx, err)
	}

	bot := data.PlatformBot{
		Platform:      platform.Kick,
		BotID:         fromDB.BotID,
		BroadcasterID: fromDB.BroadcasterID,
		UserID:        fromDB.UserID,
	}

	return bot, nil
}

func (s *BotService) BotsGet(ctx context.Context, arg data.PlatformBotsGet) ([]data.PlatformBot, error) {
	fromDB, err := s.storage.Query(ctx).KickBotsGet(ctx, db.KickBotsGetParams{
		UserID:        arg.UserID,
		BroadcasterID: arg.BroadcasterID,
		BotID:         arg.BotID,
	})
	if err != nil {
		s.logger.ErrorContext(ctx, "cannot get kick bots")
		return nil, s.storage.HandleErr(ctx, err)
	}

	var bots []data.PlatformBot
	for _, bot := range fromDB {
		bots = append(bots, data.PlatformBot{
			UserID:        bot.UserID,
			BroadcasterID: bot.BroadcasterID,
			BotID:         bot.BotID,
			Platform:      platform.Kick,
		})
	}

	return bots, nil
}

func (s *BotService) DefaultBotGet(ctx context.Context) (data.PlatformDefaultBot, error) {
	fromDB, err := s.storage.Query(ctx).KickDefaultBotGet(ctx)
	if err != nil {
		s.logger.DebugContext(ctx, "cannot get default bot")
		return data.PlatformDefaultBot{}, s.storage.HandleErr(ctx, err)
	}

	bot := data.PlatformDefaultBot{
		BotID: fromDB.BotID,
	}

	return bot, nil
}

func (s *BotService) DefaultBotChange(ctx context.Context, botID string) error {
	count, err := s.storage.Query(ctx).KickDefaultBotUpdate(ctx, botID)
	if err != nil {
		s.logger.DebugContext(ctx, "cannot update default bot", "err", err)
		return s.storage.HandleErr(ctx, err)
	}

	if count == 0 {
		s.logger.ErrorContext(ctx, "there is no default bot to update???")
		return apperror.ErrInternal
	}

	return nil
}

func (s *BotService) SelectedBotGet(ctx context.Context, userID uuid.UUID) (data.PlatformSelectedBot, error) {
	fromDB, err := s.storage.Query(ctx).KickSelectedBotGetByUserID(ctx, userID)
	if err != nil {
		s.logger.DebugContext(ctx, "cannot get selected bot")
		return data.PlatformSelectedBot{}, s.storage.HandleErr(ctx, err)
	}
	bot := data.PlatformSelectedBot{
		UserID:        fromDB.UserID,
		BotID:         fromDB.BotID,
		BroadcasterID: fromDB.BroadcasterID,
		Enabled:       fromDB.Enabled,
		UpdatedAt:     fromDB.UpdatedAt,
	}

	return bot, nil
}

func (s *BotService) SelectedBotChangeStatus(ctx context.Context, userID uuid.UUID, enable bool) error {
	_, err := s.storage.Query(ctx).KickSelectedBotStatusChange(ctx, db.KickSelectedBotStatusChangeParams{
		UserID:  userID,
		Enabled: enable,
	})
	if err != nil {
		s.logger.DebugContext(ctx, "cannot enable/disable bot", "err", err)
		return s.storage.HandleErr(ctx, err)
	}

	return nil
}

func (s *BotService) SelectedBotChange(ctx context.Context, bot data.PlatformBot) (data.PlatformSelectedBot, error) {
	fromDB, err := s.storage.Query(ctx).KickSelectedBotChange(ctx, db.KickSelectedBotChangeParams{
		UserID:        bot.UserID,
		BotID:         bot.BotID,
		BroadcasterID: bot.BroadcasterID,
	})
	if err != nil {
		s.logger.DebugContext(ctx, "cannot change selected bot", "err", err)
		return data.PlatformSelectedBot{}, s.storage.HandleErr(ctx, err)
	}

	selectedBot := data.PlatformSelectedBot{
		UserID:        fromDB.UserID,
		BroadcasterID: fromDB.BroadcasterID,
		BotID:         fromDB.BotID,
		Enabled:       fromDB.Enabled,
		UpdatedAt:     fromDB.UpdatedAt,
	}

	return selectedBot, nil
}

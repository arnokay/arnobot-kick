package service

import (
	"context"
	"log/slog"
	"strconv"

	"github.com/arnokay/arnobot-shared/apperror"
	"github.com/arnokay/arnobot-shared/applog"
	sharedData "github.com/arnokay/arnobot-shared/data"
	"github.com/arnokay/arnobot-shared/db"
	"github.com/arnokay/arnobot-shared/platform"
	sharedService "github.com/arnokay/arnobot-shared/service"
	"github.com/arnokay/arnobot-shared/storage"
	"github.com/google/uuid"

	"github.com/arnokay/arnobot-kick/internal/data"
)

type BotService struct {
	storage     storage.Storager
	txService   sharedService.ITransactionService
	authModule  *sharedService.AuthModule
	whService   *WebhookService
	kickService *KickService

	logger *slog.Logger
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

func (s *BotService) StartBot(ctx context.Context, arg sharedData.PlatformToggleBot) error {
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

  botID := strconv.Itoa(int(selectedBot.BotID))
	botProvider, err := s.authModule.AuthProviderGet(ctx, sharedData.AuthProviderGet{
		ProviderUserID: &botID,
		Provider:       platform.Kick.String(),
	})
	if err != nil {
		s.logger.ErrorContext(ctx, "cannot get bot provider")
		return err
	}

	err = s.whService.Subscribe(ctx, *botProvider, selectedBot.BroadcasterID)
	if err != nil {
		return err
	}

	s.kickService.AppSendChannelMessage(ctx, *botProvider, selectedBot.BroadcasterID, "hi!", "")

	return nil
}

func (s *BotService) StopBot(ctx context.Context, arg sharedData.PlatformToggleBot) error {
	selectedBot, err := s.SelectedBotGet(ctx, arg.UserID)
	if err != nil {
		return err
	}

  botID := strconv.Itoa(int(selectedBot.BotID))
	botProvider, err := s.authModule.AuthProviderGet(ctx, sharedData.AuthProviderGet{
		ProviderUserID: &botID,
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

	return nil
}

func (s *BotService) SelectedBotSetDefault(ctx context.Context, userID uuid.UUID) (*data.PlatformSelectedBot, error) {
	var bot *data.PlatformBot

	txCtx, err := s.txService.Begin(ctx)
	defer s.txService.Rollback(txCtx)
	if err != nil {
		return nil, err
	}

	bots, err := s.BotsGet(ctx, data.PlatformBotsGet{
		UserID: &userID,
	})
	if err != nil {
		return nil, err
	}

	if len(bots) != 0 {
		bot = &bots[0]
	} else {
		defaultBot, err := s.DefaultBotGet(ctx)
		if err != nil {
			return nil, err
		}
		userProvider, err := s.authModule.AuthProviderGet(ctx, sharedData.AuthProviderGet{
			UserID:   &userID,
			Provider: platform.Kick.String(),
		})
		if err != nil {
			return nil, err
		}
    providerUserID, err := strconv.Atoi(userProvider.ProviderUserID)
    if err != nil {
      s.logger.ErrorContext(ctx, "cant parse providerUserID to string", "err", err, "providerUserID", providerUserID)
    }
		bot, err = s.BotCreate(ctx, data.PlatformBotCreate{
			UserID:        userID,
			BotID:         defaultBot.BotID,
			BroadcasterID: int32(providerUserID),
		})
		if err != nil {
			return nil, err
		}
	}

	selectedBot, err := s.SelectedBotChange(ctx, *bot)
	if err != nil {
		return nil, err
	}

	err = s.txService.Commit(txCtx)
	if err != nil {
		return nil, err
	}

	return selectedBot, nil
}

func (s *BotService) SelectedBotGetByBroadcasterID(ctx context.Context, broadcasterID int32) (*data.PlatformSelectedBot, error) {
	fromDB, err := s.storage.Query(ctx).KickSelectedBotGetByBroadcasterID(ctx, broadcasterID)
	if err != nil {
		s.logger.DebugContext(ctx, "cannot get selected bot", "err", err, "broadcasterID", broadcasterID)
		return nil, s.storage.HandleErr(ctx, err)
	}

	bot := data.NewPlatformSelectedBotFromDB(fromDB)

	return &bot, nil
}

func (s *BotService) BotCreate(ctx context.Context, arg data.PlatformBotCreate) (*data.PlatformBot, error) {
	fromDB, err := s.storage.Query(ctx).KickBotCreate(ctx, arg.ToDB())
	if err != nil {
		s.logger.DebugContext(ctx, "cannot create bot", "err", err)
		return nil, s.storage.HandleErr(ctx, err)
	}

	bot := data.NewPlatformBotFromDB(fromDB)

	return &bot, nil
}

func (s *BotService) BotsGet(ctx context.Context, arg data.PlatformBotsGet) ([]data.PlatformBot, error) {
	fromDB, err := s.storage.Query(ctx).KickBotsGet(ctx, arg.ToDB())
	if err != nil {
		s.logger.ErrorContext(ctx, "cannot get kick bots")
		return nil, s.storage.HandleErr(ctx, err)
	}

	var bots []data.PlatformBot
	for _, bot := range fromDB {
		bots = append(bots, data.NewPlatformBotFromDB(bot))
	}

	return bots, nil
}

func (s *BotService) DefaultBotGet(ctx context.Context) (*data.PlatformDefaultBot, error) {
	fromDB, err := s.storage.Query(ctx).KickDefaultBotGet(ctx)
	if err != nil {
		s.logger.DebugContext(ctx, "cannot get default bot")
		return nil, s.storage.HandleErr(ctx, err)
	}
	bot := data.NewPlatformDefaultBotFromDB(fromDB)

	return &bot, nil
}

func (s *BotService) DefaultBotChange(ctx context.Context, botID int32) error {
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

func (s *BotService) SelectedBotGet(ctx context.Context, userID uuid.UUID) (*data.PlatformSelectedBot, error) {
	fromDB, err := s.storage.Query(ctx).KickSelectedBotGetByUserID(ctx, userID)
	if err != nil {
		s.logger.DebugContext(ctx, "cannot get selected bot")
		return nil, s.storage.HandleErr(ctx, err)
	}
	bot := data.NewPlatformSelectedBotFromDB(fromDB)

	return &bot, nil
}

func (s *BotService) SelectedBotChange(ctx context.Context, bot data.PlatformBot) (*data.PlatformSelectedBot, error) {
	fromDB, err := s.storage.Query(ctx).KickSelectedBotChange(ctx, db.KickSelectedBotChangeParams{
		UserID:        bot.UserID,
		BotID:         bot.BotID,
		BroadcasterID: bot.BroadcasterID,
	})
	if err != nil {
		s.logger.DebugContext(ctx, "cannot change selected bot", "err", err)
		return nil, s.storage.HandleErr(ctx, err)
	}

	selectedBot := data.NewPlatformSelectedBotFromDB(fromDB)

	return &selectedBot, nil
}

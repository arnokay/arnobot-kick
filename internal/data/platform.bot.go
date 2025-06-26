package data

import (
	"time"

	"github.com/arnokay/arnobot-shared/db"
	"github.com/google/uuid"
)

type PlatformDefaultBot struct {
	BotID int32
}

func NewPlatformDefaultBotFromDB(fromDB db.KickDefaultBot) PlatformDefaultBot {
	return PlatformDefaultBot{
		BotID: fromDB.BotID,
	}
}

type PlatformSelectedBot struct {
	UserID        uuid.UUID
	BotID         int32
	BroadcasterID int32
	UpdatedAt     time.Time
}

func NewPlatformSelectedBotFromDB(fromDB db.KickSelectedBot) PlatformSelectedBot {
	return PlatformSelectedBot{
		UserID:        fromDB.UserID,
		BotID:         fromDB.BotID,
		BroadcasterID: fromDB.BroadcasterID,
	}
}

type PlatformBot struct {
	UserID        uuid.UUID
	BotID         int32
	BroadcasterID int32
}

func NewPlatformBotFromDB(fromDB db.KickBot) PlatformBot {
	return PlatformBot{
		UserID:        fromDB.UserID,
		BroadcasterID: fromDB.BroadcasterID,
		BotID:         fromDB.BotID,
	}
}

type PlatformBotCreate struct {
	UserID        uuid.UUID
	BotID         int32
	BroadcasterID int32
}

func (d PlatformBotCreate) ToDB() db.KickBotCreateParams {
	return db.KickBotCreateParams{
		UserID:        d.UserID,
		BroadcasterID: d.BroadcasterID,
		BotID:         d.BotID,
	}
}

type PlatformBotsGet struct {
	UserID        *uuid.UUID
	BotID         *int32
	BroadcasterID *int32
}

func (d PlatformBotsGet) ToDB() db.KickBotsGetParams {
	return db.KickBotsGetParams{
		UserID:        d.UserID,
		BotID:         d.BotID,
		BroadcasterID: d.BroadcasterID,
	}
}

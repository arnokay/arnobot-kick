package data

import (
	"time"

	"github.com/arnokay/arnobot-shared/db"
	"github.com/google/uuid"
)

type PlatformDefaultBot struct {
	BotID int
}

func NewPlatformDefaultBotFromDB(fromDB db.KickDefaultBot) PlatformDefaultBot {
	return PlatformDefaultBot{
		BotID: int(fromDB.BotID),
	}
}

type PlatformSelectedBot struct {
	UserID        uuid.UUID
	BotID         int
	BroadcasterID int
	Enabled       bool
	UpdatedAt     time.Time
}

func NewPlatformSelectedBotFromDB(fromDB db.KickSelectedBot) PlatformSelectedBot {
	return PlatformSelectedBot{
		UserID:        fromDB.UserID,
		BotID:         int(fromDB.BotID),
		BroadcasterID: int(fromDB.BroadcasterID),
		Enabled:       fromDB.Enabled,
	}
}

type PlatformBot struct {
	UserID        uuid.UUID
	BotID         int
	BroadcasterID int
}

func NewPlatformBotFromDB(fromDB db.KickBot) PlatformBot {
	return PlatformBot{
		UserID:        fromDB.UserID,
		BroadcasterID: int(fromDB.BroadcasterID),
		BotID:         int(fromDB.BotID),
	}
}

type PlatformBotCreate struct {
	UserID        uuid.UUID
	BotID         int
	BroadcasterID int
}

func (d PlatformBotCreate) ToDB() db.KickBotCreateParams {
	return db.KickBotCreateParams{
		UserID:        d.UserID,
		BroadcasterID: int32(d.BroadcasterID),
		BotID:         int32(d.BotID),
	}
}

type PlatformBotsGet struct {
	UserID        *uuid.UUID
	BotID         *int
	BroadcasterID *int
}

func (d PlatformBotsGet) ToDB() db.KickBotsGetParams {
	var botID int32
	if d.BotID != nil {
		botID = int32(*d.BotID)
	}
	var broadcasterID int32
	if d.BroadcasterID != nil {
		broadcasterID = int32(*d.BroadcasterID)
	}
	return db.KickBotsGetParams{
		UserID:        d.UserID,
		BotID:         &botID,
		BroadcasterID: &broadcasterID,
	}
}

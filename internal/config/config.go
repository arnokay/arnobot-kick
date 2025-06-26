package config

import (
	"flag"
	"fmt"
	"os"
	"strconv"

	"github.com/arnokay/arnobot-shared/pkg/assert"
)

const (
	EnvMBURL            = "MB_URL"
	EnvDBDsn            = "DB_DSN"
	EnvKickWHSecret     = "KICK_WH_SECRET"
	EnvKickWHCallback   = "KICK_WH_CALLBACK"
	EnvPort             = "PORT"
	EnvKickClientID     = "KICK_CLIENT_ID"
	EnvKickClientSecret = "KICK_CLIENT_SECRET"
)

type config struct {
	Global   GlobalConfig
	Kick     KickConfig
	MB       MBConfig
	DB       DBConfig
	Webhooks Webhooks
}

type KickConfig struct {
	ClientID     string
	ClientSecret string
}

type DBConfig struct {
	DSN          string
	MaxIdleConns int
	MaxOpenConns int
	MaxIdleTime  string
}

type GlobalConfig struct {
	LogLevel int
	Port     int
}

type MBConfig struct {
	URL string
}

type Webhooks struct {
	Secret   string
	Callback string
}

var Config *config

func Load() *config {
	Config = &config{
		Global: GlobalConfig{
			LogLevel: -4,
		},
	}

	if os.Getenv(EnvPort) != "" {
		port, err := strconv.Atoi(os.Getenv(EnvPort))
		assert.NoError(err, fmt.Sprintf("%v: not a number", EnvPort))
		Config.Global.Port = port
	}

	flag.StringVar(&Config.MB.URL, "mb-url", os.Getenv(EnvMBURL), "Message Broker URL")
	flag.IntVar(&Config.Global.LogLevel, "log-level", Config.Global.LogLevel, "Minimal Log Level (default: -4)")
	flag.StringVar(&Config.Webhooks.Secret, "wh-secret", os.Getenv(EnvKickWHSecret), "secret for subscribing to webhooks")
	flag.StringVar(&Config.Webhooks.Callback, "wh-callback", os.Getenv(EnvKickWHCallback), "kick secret")
	flag.IntVar(&Config.Global.Port, "port", Config.Global.Port, "http port")
	flag.StringVar(&Config.Kick.ClientID, "client-id", os.Getenv(EnvKickClientID), "kick client id")
	flag.StringVar(&Config.Kick.ClientSecret, "client-secret", os.Getenv(EnvKickClientSecret), "kick client id")
	flag.StringVar(&Config.DB.DSN, "db-dsn", os.Getenv(EnvDBDsn), "DB DSN")
	flag.IntVar(&Config.DB.MaxIdleConns, "db-max-idle-conns", 25, "PostgreSQL max idle connections")
	flag.IntVar(&Config.DB.MaxOpenConns, "db-max-open-conns", 25, "PostgreSQL max open connections")
	flag.StringVar(&Config.DB.MaxIdleTime, "db-max-idle-time", "15m", "PostgreSQL max connection idle time")

	flag.Parse()

	return Config
}

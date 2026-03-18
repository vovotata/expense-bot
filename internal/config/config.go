package config

import (
	"fmt"
	"time"

	"github.com/kelseyhightower/envconfig"
)

type Config struct {
	BotToken       string  `envconfig:"BOT_TOKEN" required:"true"`
	AdminChatID    int64   `envconfig:"ADMIN_CHAT_ID" required:"true"`
	AllowedUserIDs []int64 `envconfig:"ALLOWED_USER_IDS"`

	DBHost            string        `envconfig:"DB_HOST" default:"localhost"`
	DBPort            int           `envconfig:"DB_PORT" default:"5432"`
	DBUser            string        `envconfig:"DB_USER" default:"bot"`
	DBPassword        string        `envconfig:"DB_PASSWORD" required:"true"`
	DBName            string        `envconfig:"DB_NAME" default:"expense_bot"`
	DBSSLMode         string        `envconfig:"DB_SSL_MODE" default:"disable"`
	DBMaxOpenConns    int           `envconfig:"DB_MAX_OPEN_CONNS" default:"25"`
	DBMaxIdleConns    int           `envconfig:"DB_MAX_IDLE_CONNS" default:"5"`
	DBConnMaxLifetime time.Duration `envconfig:"DB_CONN_MAX_LIFETIME" default:"5m"`

	LogLevel        string        `envconfig:"LOG_LEVEL" default:"info"`
	FSMTTL          time.Duration `envconfig:"FSM_TTL" default:"30m"`
	RateLimitPerMin int           `envconfig:"RATE_LIMIT_PER_MIN" default:"30"`

	EmailEncryptionKey      string        `envconfig:"EMAIL_ENCRYPTION_KEY"`
	EmailIDLETimeout        time.Duration `envconfig:"EMAIL_IDLE_TIMEOUT" default:"25m"`
	EmailReconnectMaxBackoff time.Duration `envconfig:"EMAIL_RECONNECT_MAX_BACKOFF" default:"60s"`
	EmailCodeTTL            time.Duration `envconfig:"EMAIL_CODE_TTL" default:"24h"`
	EmailMaxAccountsPerUser int           `envconfig:"EMAIL_MAX_ACCOUNTS_PER_USER" default:"5"`
}

func Load() (*Config, error) {
	var cfg Config
	if err := envconfig.Process("", &cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}

func (c *Config) DatabaseURL() string {
	return "postgres://" + c.DBUser + ":" + c.DBPassword + "@" + c.DBHost + ":" + itoa(c.DBPort) + "/" + c.DBName + "?sslmode=" + c.DBSSLMode
}

func itoa(i int) string {
	return fmt.Sprintf("%d", i)
}

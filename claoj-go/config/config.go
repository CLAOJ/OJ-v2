package config

import (
	"log"
	"strings"

	"github.com/spf13/viper"
)

// Config holds all application configuration.
type Config struct {
	Server   ServerConfig   `mapstructure:"server"`
	Database DatabaseConfig `mapstructure:"database"`
	Redis    RedisConfig    `mapstructure:"redis"`
	App      AppConfig      `mapstructure:"app"`
	Email    EmailConfig    `mapstructure:"email"`
	OAuth    OAuthConfig    `mapstructure:"oauth"`
}

type ServerConfig struct {
	Port string `mapstructure:"port"`
	Mode string `mapstructure:"mode"` // gin mode: debug | release
}

type DatabaseConfig struct {
	DSN string `mapstructure:"dsn"` // full MySQL DSN, e.g. user:pass@tcp(host:3306)/db?charset=utf8mb4&parseTime=True&loc=UTC
}

type RedisConfig struct {
	Addr     string `mapstructure:"addr"`
	Password string `mapstructure:"password"`
	DB       int    `mapstructure:"db"`
}

type AppConfig struct {
	SecretKey          string `mapstructure:"secret_key"`
	SiteFullURL        string `mapstructure:"site_full_url"`
	EventDaemonSubmKey string `mapstructure:"event_daemon_subm_key"`
	EventDaemonContKey string `mapstructure:"event_daemon_cont_key"`
	DefaultLanguage    string `mapstructure:"default_language"`
}

type EmailConfig struct {
	SMTPHost     string `mapstructure:"smtp_host"`
	SMTPPort     int    `mapstructure:"smtp_port"`
	SMTPUser     string `mapstructure:"smtp_user"`
	SMTPPassword string `mapstructure:"smtp_password"`
	FromEmail    string `mapstructure:"from_email"`
	FromName     string `mapstructure:"from_name"`
	NoReply      bool   `mapstructure:"no_reply"` // if true, emails won't be sent (for development)
}

type OAuthConfig struct {
	Google OAuthProviderConfig `mapstructure:"google"`
	GitHub OAuthProviderConfig `mapstructure:"github"`
}

type OAuthProviderConfig struct {
	ClientID     string   `mapstructure:"client_id"`
	ClientSecret string   `mapstructure:"client_secret"`
	RedirectURL  string   `mapstructure:"redirect_url"`
	Enabled      bool     `mapstructure:"enabled"`
	Scopes       []string `mapstructure:"scopes"`
}

var C Config

// Load reads configuration from environment variables and/or a config file.
// Environment variables take precedence. Prefix: CLAOJ_
// Example:
//
//	CLAOJ_DATABASE_DSN=user:pass@tcp(127.0.0.1:3306)/claoj?parseTime=True
//	CLAOJ_REDIS_ADDR=127.0.0.1:6379
//	CLAOJ_SERVER_PORT=8081
func Load() {
	v := viper.New()

	// defaults
	v.SetDefault("server.port", "8081")
	v.SetDefault("server.mode", "debug")
	v.SetDefault("database.dsn", "")
	v.SetDefault("redis.addr", "127.0.0.1:6379")
	v.SetDefault("redis.password", "")
	v.SetDefault("redis.db", 0)
	v.SetDefault("app.secret_key", "changeme")
	v.SetDefault("app.site_full_url", "http://localhost:8081")
	v.SetDefault("app.event_daemon_subm_key", "")
	v.SetDefault("app.event_daemon_cont_key", "")
	v.SetDefault("app.default_language", "py3")
	v.SetDefault("email.smtp_host", "")
	v.SetDefault("email.smtp_port", 587)
	v.SetDefault("email.smtp_user", "")
	v.SetDefault("email.smtp_password", "")
	v.SetDefault("email.from_email", "noreply@claoj.edu.vn")
	v.SetDefault("email.from_name", "CLAOJ")
	v.SetDefault("email.no_reply", true)

	// OAuth defaults
	v.SetDefault("oauth.google.client_id", "")
	v.SetDefault("oauth.google.client_secret", "")
	v.SetDefault("oauth.google.redirect_url", "")
	v.SetDefault("oauth.google.enabled", false)
	v.SetDefault("oauth.github.client_id", "")
	v.SetDefault("oauth.github.client_secret", "")
	v.SetDefault("oauth.github.redirect_url", "")
	v.SetDefault("oauth.github.enabled", false)

	// config file (optional)
	v.SetConfigName("claoj")
	v.SetConfigType("yaml")
	v.AddConfigPath(".")
	v.AddConfigPath("./config")
	_ = v.ReadInConfig() // no-error if not found

	// bind environment variables (env takes precedence over config file)
	v.SetEnvPrefix("CLAOJ")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.BindEnv("app.secret_key", "CLAOJ_APP_SECRET_KEY")
	v.BindEnv("database.dsn", "CLAOJ_DATABASE_DSN")
	v.BindEnv("redis.addr", "CLAOJ_REDIS_ADDR")
	v.BindEnv("server.port", "CLAOJ_SERVER_PORT")
	v.AutomaticEnv()

	if err := v.Unmarshal(&C); err != nil {
		log.Fatalf("config: failed to unmarshal: %v", err)
	}

	// Security validation
	if C.App.SecretKey == "" || C.App.SecretKey == "changeme" || C.App.SecretKey == "<GENERATE_SECURE_KEY_ON_DEPLOY>" {
		log.Fatal("config: FATAL - app.secret_key is not set or using default value. Generate a secure key using: openssl rand -base64 64")
	}

	if C.Database.DSN == "" {
		log.Println("config: WARNING — CLAOJ_DATABASE_DSN is not set; DB will not connect")
	}
}

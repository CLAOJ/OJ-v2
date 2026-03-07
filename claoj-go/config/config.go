package config

import (
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/spf13/viper"
	"github.com/subosito/gotenv"
)

// readSecretFromFile reads a secret from a Docker secret file
func readSecretFromFile(filePath string) string {
	if filePath == "" {
		return ""
	}
	data, err := os.ReadFile(filePath)
	if err != nil {
		log.Printf("config: warning: failed to read secret file %s: %v", filePath, err)
		return ""
	}
	return strings.TrimSpace(string(data))
}

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
	SecretKey            string `mapstructure:"secret_key"`
	JwtSecretKey         string `mapstructure:"jwt_secret_key"`
	RequireTotpForAdmins bool   `mapstructure:"require_totp_for_admins"`
	SiteFullURL          string `mapstructure:"site_full_url"`
	DefaultLanguage      string `mapstructure:"default_language"`
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

// Load reads configuration from environment variables.
//
// Priority (highest to lowest):
//  1. Direct environment variables (DATABASE_URL, SECRET_KEY, etc.)
//  2. Prefixed environment variables (CLAOJ_DATABASE_DSN, CLAOJ_APP_SECRET_KEY)
//  3. .env file (loaded via godotenv)
//  4. Default values
//
// Example:
//
//	DATABASE_URL=user:pass@tcp(127.0.0.1:3306)/claoj?parseTime=True
//	REDIS_URL=127.0.0.1:6379
//	SECRET_KEY=your-secret-key
//	SITE_URL=http://localhost:3000
func Load() {
	// Load .env file if it exists (optional)
	if _, err := os.Stat(".env"); err == nil {
		if err := gotenv.Load(".env"); err != nil {
			log.Printf("config: warning: failed to load .env file: %v", err)
		}
	} else if _, err := os.Stat("../.env"); err == nil {
		// Support loading from parent directory (for Docker)
		if err := gotenv.Load("../.env"); err != nil {
			log.Printf("config: warning: failed to load ../.env file: %v", err)
		}
	}

	v := viper.New()

	// defaults
	v.SetDefault("server.port", "8081")
	v.SetDefault("server.mode", "debug")
	v.SetDefault("database.dsn", "")
	v.SetDefault("redis.addr", "127.0.0.1:6379")
	v.SetDefault("redis.password", "")
	v.SetDefault("redis.db", 0)
	v.SetDefault("app.secret_key", "changeme")
	v.SetDefault("app.jwt_secret_key", "") // Defaults to secret_key if not set
	v.SetDefault("app.require_totp_for_admins", false)
	v.SetDefault("app.site_full_url", "http://localhost:8081")
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

	// bind environment variables - support both plain names and CLAOJ_* prefix
	v.SetEnvPrefix("CLAOJ")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	// Plain environment variable names (no prefix) - these are checked first
	v.BindEnv("database.dsn", "DATABASE_DSN", "DATABASE_URL", "CLAOJ_DATABASE_DSN")
	v.BindEnv("redis.addr", "REDIS_ADDR", "REDIS_URL", "CLAOJ_REDIS_ADDR")
	v.BindEnv("redis.password", "REDIS_PASSWORD", "CLAOJ_REDIS_PASSWORD")
	v.BindEnv("redis.db", "REDIS_DB", "CLAOJ_REDIS_DB")
	v.BindEnv("server.port", "SERVER_PORT", "CLAOJ_SERVER_PORT")
	v.BindEnv("server.mode", "SERVER_MODE", "CLAOJ_SERVER_MODE")
	v.BindEnv("app.secret_key", "SECRET_KEY", "CLAOJ_APP_SECRET_KEY")
	v.BindEnv("app.jwt_secret_key", "JWT_SECRET_KEY", "CLAOJ_JWT_SECRET_KEY")
	v.BindEnv("app.require_totp_for_admins", "REQUIRE_TOTP_FOR_ADMINS", "CLAOJ_REQUIRE_TOTP_FOR_ADMINS")
	v.BindEnv("app.site_full_url", "SITE_URL", "SITE_FULL_URL", "CLAOJ_SITE_FULL_URL")
	v.BindEnv("app.default_language", "DEFAULT_LANG", "DEFAULT_LANGUAGE", "CLAOJ_DEFAULT_LANGUAGE")

	// Email
	v.BindEnv("email.smtp_host", "SMTP_HOST", "CLAOJ_SMTP_HOST")
	v.BindEnv("email.smtp_port", "SMTP_PORT", "CLAOJ_SMTP_PORT")
	v.BindEnv("email.smtp_user", "SMTP_USER", "CLAOJ_SMTP_USER")
	v.BindEnv("email.smtp_password", "SMTP_PASSWORD", "CLAOJ_SMTP_PASSWORD")
	v.BindEnv("email.from_email", "FROM_EMAIL", "CLAOJ_FROM_EMAIL")
	v.BindEnv("email.from_name", "FROM_NAME", "CLAOJ_FROM_NAME")
	v.BindEnv("email.no_reply", "EMAIL_NO_REPLY", "CLAOJ_EMAIL_NO_REPLY")

	// OAuth - Google
	v.BindEnv("oauth.google.client_id", "OAUTH_GOOGLE_CLIENT_ID", "GOOGLE_CLIENT_ID", "CLAOJ_OAUTH_GOOGLE_CLIENT_ID")
	v.BindEnv("oauth.google.client_secret", "OAUTH_GOOGLE_CLIENT_SECRET", "GOOGLE_CLIENT_SECRET", "CLAOJ_OAUTH_GOOGLE_CLIENT_SECRET")
	v.BindEnv("oauth.google.redirect_url", "OAUTH_GOOGLE_REDIRECT_URL", "GOOGLE_REDIRECT_URL", "CLAOJ_OAUTH_GOOGLE_REDIRECT_URL")
	v.BindEnv("oauth.google.enabled", "OAUTH_GOOGLE_ENABLED", "CLAOJ_OAUTH_GOOGLE_ENABLED")

	// OAuth - GitHub
	v.BindEnv("oauth.github.client_id", "OAUTH_GITHUB_CLIENT_ID", "GITHUB_CLIENT_ID", "CLAOJ_OAUTH_GITHUB_CLIENT_ID")
	v.BindEnv("oauth.github.client_secret", "OAUTH_GITHUB_CLIENT_SECRET", "GITHUB_CLIENT_SECRET", "CLAOJ_OAUTH_GITHUB_CLIENT_SECRET")
	v.BindEnv("oauth.github.redirect_url", "OAUTH_GITHUB_REDIRECT_URL", "GITHUB_REDIRECT_URL", "CLAOJ_OAUTH_GITHUB_REDIRECT_URL")
	v.BindEnv("oauth.github.enabled", "OAUTH_GITHUB_ENABLED", "CLAOJ_OAUTH_GITHUB_ENABLED")

	// Secret file paths (for Docker secrets)
	v.BindEnv("app.secret_key_file", "SECRET_KEY_FILE", "CLAOJ_SECRET_KEY_FILE")
	v.BindEnv("app.jwt_secret_key_file", "JWT_SECRET_KEY_FILE", "CLAOJ_JWT_SECRET_KEY_FILE")
	v.BindEnv("database.password_file", "MYSQL_PASSWORD_FILE", "CLAOJ_MYSQL_PASSWORD_FILE")
	// Database components for DSN construction
	v.BindEnv("database.host", "MYSQL_HOST", "MYSQL_HOST_ENV", "CLAOJ_MYSQL_HOST")
	v.BindEnv("database.port", "MYSQL_PORT", "MYSQL_PORT_ENV", "CLAOJ_MYSQL_PORT")
	v.BindEnv("database.user", "MYSQL_USER", "MYSQL_USER_ENV", "CLAOJ_MYSQL_USER")
	v.BindEnv("database.name", "MYSQL_DATABASE", "MYSQL_DATABASE_ENV", "CLAOJ_MYSQL_DATABASE")

	v.AutomaticEnv()

	if err := v.Unmarshal(&C); err != nil {
		log.Fatalf("config: failed to unmarshal: %v", err)
	}

	// Load secrets from Docker secret files if paths are provided
	// Priority: Secret file > Environment variable > Default
	if secretKeyFile := v.GetString("app.secret_key_file"); secretKeyFile != "" {
		if secret := readSecretFromFile(secretKeyFile); secret != "" {
			C.App.SecretKey = secret
			log.Println("config: loaded secret_key from Docker secret file")
		}
	}
	if jwtSecretKeyFile := v.GetString("app.jwt_secret_key_file"); jwtSecretKeyFile != "" {
		if secret := readSecretFromFile(jwtSecretKeyFile); secret != "" {
			C.App.JwtSecretKey = secret
			log.Println("config: loaded jwt_secret_key from Docker secret file")
		}
	}
	if dbPasswordFile := v.GetString("database.password_file"); dbPasswordFile != "" {
		if password := readSecretFromFile(dbPasswordFile); password != "" {
			log.Println("config: loaded database password from Docker secret file")
			// Construct DSN from components using password from secret file
			host := v.GetString("database.host")
			port := v.GetString("database.port")
			user := v.GetString("database.user")
			dbname := v.GetString("database.name")
			if host != "" && user != "" && dbname != "" {
				C.Database.DSN = fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?charset=utf8mb4&parseTime=True&loc=UTC", user, password, host, port, dbname)
				log.Println("config: constructed DSN from Docker secret-based credentials")
			}
		}
	}

	// Security validation
	if C.App.SecretKey == "" || C.App.SecretKey == "changeme" || C.App.SecretKey == "<GENERATE_SECURE_KEY_ON_DEPLOY>" {
		log.Fatal("config: FATAL - app.secret_key is not set or using default value. Generate a secure key using: openssl rand -base64 64")
	}

	// JWT secret key validation - use SecretKey as fallback if JWT secret is not set
	if C.App.JwtSecretKey == "" {
		log.Println("config: WARNING - JWT_SECRET_KEY is not set, using SECRET_KEY instead. For better security, set a dedicated JWT_SECRET_KEY.")
		C.App.JwtSecretKey = C.App.SecretKey
	}

	if C.Database.DSN == "" {
		log.Println("config: WARNING — DATABASE_DSN is not set; DB will not connect")
	}
}

package config

import (
	"fmt"
	"net"
	"net/url"
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

type Config struct {
	AppPort string
	AppEnv  string

	// Database
	DatabaseURL    string
	DBHost         string
	DBPort         string
	DBUser         string
	DBPassword     string
	DBName         string
	DBSSLMode      string
	DBMaxOpenConns int
	DBMaxIdleConns int

	// Redis
	RedisURL      string
	RedisAddr     string
	RedisPassword string
	RedisDB       int

	// Security
	AESSecretKey              []byte
	HMACSecretKey             []byte
	HMACTimestampTolerance    int64
	WebhookSecretKey          []byte
	WebhookTimestampTolerance int64
	IdempotencyTTLSeconds     int

	// AI Social Commerce & Queue Worker
	GeminiAPIKey         string
	GeminiModel          string
	EnableEmbeddedWorker bool
}

func (c *Config) GetDSN() string {
	if c.DatabaseURL != "" {
		return c.DatabaseURL
	}
	u := &url.URL{
		Scheme:   "postgres",
		User:     url.UserPassword(c.DBUser, c.DBPassword),
		Host:     net.JoinHostPort(c.DBHost, c.DBPort),
		Path:     c.DBName,
		RawQuery: "sslmode=" + url.QueryEscape(c.DBSSLMode),
	}
	return u.String()
}

func LoadConfig() (*Config, error) {
	_ = godotenv.Load()

	aesKey := getEnv("AES_SECRET_KEY", "01234567890123456789012345678901")
	if len(aesKey) != 32 {
		return nil, fmt.Errorf("config error: AES_SECRET_KEY must be exactly 32 bytes (got %d)", len(aesKey))
	}

	hmacKey := getEnv("HMAC_SECRET_KEY", "spg-secret-hmac-key-for-signatures")
	webhookSecret := getEnv("WEBHOOK_SECRET", "spg-webhook-secret-hmac-2026")

	maxOpenConns, _ := strconv.Atoi(getEnv("DB_MAX_OPEN_CONNS", "25"))
	maxIdleConns, _ := strconv.Atoi(getEnv("DB_MAX_IDLE_CONNS", "10"))
	redisDB, _ := strconv.Atoi(getEnv("REDIS_DB", "0"))
	timestampTolerance, _ := strconv.ParseInt(getEnv("HMAC_TIMESTAMP_TOLERANCE_SECONDS", "300"), 10, 64)
	webhookTolerance, _ := strconv.ParseInt(getEnv("WEBHOOK_TIMESTAMP_TOLERANCE_SECONDS", "300"), 10, 64)
	idempotencyTTL, _ := strconv.Atoi(getEnv("IDEMPOTENCY_TTL_SECONDS", "120"))
	enableWorker := getEnv("ENABLE_EMBEDDED_WORKER", "true") != "false"

	appPort := getEnv("APP_PORT", "")
	if appPort == "" {
		appPort = getEnv("PORT", "8080")
	}

	return &Config{
		AppPort:                   appPort,
		AppEnv:                    getEnv("APP_ENV", "development"),
		DatabaseURL:               getEnv("DATABASE_URL", ""),
		DBHost:                    getEnv("DB_HOST", "localhost"),
		DBPort:                    getEnv("DB_PORT", "5432"),
		DBUser:                    getEnv("DB_USER", "postgres"),
		DBPassword:                getEnv("DB_PASSWORD", "postgres"),
		DBName:                    getEnv("DB_NAME", "secure_payment_gateway"),
		DBSSLMode:                 getEnv("DB_SSLMODE", "disable"),
		DBMaxOpenConns:            maxOpenConns,
		DBMaxIdleConns:            maxIdleConns,
		RedisURL:                  getEnv("REDIS_URL", ""),
		RedisAddr:                 getEnv("REDIS_ADDR", "localhost:6379"),
		RedisPassword:             getEnv("REDIS_PASSWORD", ""),
		RedisDB:                   redisDB,
		AESSecretKey:              []byte(aesKey),
		HMACSecretKey:             []byte(hmacKey),
		HMACTimestampTolerance:    timestampTolerance,
		WebhookSecretKey:          []byte(webhookSecret),
		WebhookTimestampTolerance: webhookTolerance,
		IdempotencyTTLSeconds:     idempotencyTTL,
		GeminiAPIKey:              getEnv("GEMINI_API_KEY", ""),
		GeminiModel:               getEnv("GEMINI_MODEL", "gemini-2.0-flash"),
		EnableEmbeddedWorker:      enableWorker,
	}, nil
}

func getEnv(key, defaultVal string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultVal
}


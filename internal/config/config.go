package config

import (
	"fmt"
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

type Config struct {
	AppPort string
	AppEnv  string

	// Database
	DBHost         string
	DBPort         string
	DBUser         string
	DBPassword     string
	DBName         string
	DBSSLMode      string
	DBMaxOpenConns int
	DBMaxIdleConns int

	// Redis
	RedisAddr     string
	RedisPassword string
	RedisDB       int

	// Security
	AESSecretKey           []byte
	HMACSecretKey          []byte
	HMACTimestampTolerance int64
	IdempotencyTTLSeconds  int
}

func (c *Config) GetDSN() string {
	return fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=%s",
		c.DBUser, c.DBPassword, c.DBHost, c.DBPort, c.DBName, c.DBSSLMode,
	)
}

func LoadConfig() (*Config, error) {
	_ = godotenv.Load()

	aesKey := getEnv("AES_SECRET_KEY", "01234567890123456789012345678901")
	if len(aesKey) != 32 {
		return nil, fmt.Errorf("config error: AES_SECRET_KEY must be exactly 32 bytes (got %d)", len(aesKey))
	}

	hmacKey := getEnv("HMAC_SECRET_KEY", "spg-secret-hmac-key-for-signatures")

	maxOpenConns, _ := strconv.Atoi(getEnv("DB_MAX_OPEN_CONNS", "25"))
	maxIdleConns, _ := strconv.Atoi(getEnv("DB_MAX_IDLE_CONNS", "10"))
	redisDB, _ := strconv.Atoi(getEnv("REDIS_DB", "0"))
	timestampTolerance, _ := strconv.ParseInt(getEnv("HMAC_TIMESTAMP_TOLERANCE_SECONDS", "300"), 10, 64)
	idempotencyTTL, _ := strconv.Atoi(getEnv("IDEMPOTENCY_TTL_SECONDS", "120"))

	return &Config{
		AppPort:                getEnv("APP_PORT", "8080"),
		AppEnv:                 getEnv("APP_ENV", "development"),
		DBHost:                 getEnv("DB_HOST", "localhost"),
		DBPort:                 getEnv("DB_PORT", "5432"),
		DBUser:                 getEnv("DB_USER", "postgres"),
		DBPassword:             getEnv("DB_PASSWORD", "postgres"),
		DBName:                 getEnv("DB_NAME", "secure_payment_gateway"),
		DBSSLMode:              getEnv("DB_SSLMODE", "disable"),
		DBMaxOpenConns:         maxOpenConns,
		DBMaxIdleConns:         maxIdleConns,
		RedisAddr:              getEnv("REDIS_ADDR", "localhost:6379"),
		RedisPassword:          getEnv("REDIS_PASSWORD", ""),
		RedisDB:                redisDB,
		AESSecretKey:           []byte(aesKey),
		HMACSecretKey:          []byte(hmacKey),
		HMACTimestampTolerance: timestampTolerance,
		IdempotencyTTLSeconds:  idempotencyTTL,
	}, nil
}

func getEnv(key, defaultVal string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultVal
}

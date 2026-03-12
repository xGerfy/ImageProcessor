package config

import (
	"github.com/wb-go/wbf/config"
)

// Config - конфигурация приложения
type Config struct {
	ServerAddr    string
	ServerPort    int
	LogLevel      string
	KafkaBrokers  []string
	KafkaTopic    string
	KafkaGroupID  string
	RedisAddr     string
	RedisPassword string
	StorageOrig   string
	StorageProc   string
	WatermarkText string
	ThumbWidth    int
	ThumbHeight   int
	ResizeWidth   int
	ResizeHeight  int
}

// Load загружает конфигурацию из .env
func Load() (*Config, error) {
	cfg := config.New()

	// Загрузка .env файлов
	if err := cfg.LoadEnvFiles("./.env"); err != nil {
		return nil, err
	}

	// Включение переменных окружения
	cfg.EnableEnv("")

	return &Config{
		ServerAddr:    cfg.GetString("SERVER_ADDR"),
		ServerPort:    cfg.GetInt("SERVER_PORT"),
		LogLevel:      cfg.GetString("LOG_LEVEL"),
		KafkaBrokers:  []string{cfg.GetString("KAFKA_BROKERS")},
		KafkaTopic:    cfg.GetString("KAFKA_TOPIC"),
		KafkaGroupID:  cfg.GetString("KAFKA_CONSUMER_GROUP"),
		RedisAddr:     cfg.GetString("REDIS_ADDR"),
		RedisPassword: cfg.GetString("REDIS_PASSWORD"),
		StorageOrig:   cfg.GetString("STORAGE_ORIGINALS_PATH"),
		StorageProc:   cfg.GetString("STORAGE_PROCESSED_PATH"),
		WatermarkText: cfg.GetString("WATERMARK_TEXT"),
		ThumbWidth:    cfg.GetInt("THUMBNAIL_WIDTH"),
		ThumbHeight:   cfg.GetInt("THUMBNAIL_HEIGHT"),
		ResizeWidth:   cfg.GetInt("RESIZE_WIDTH"),
		ResizeHeight:  cfg.GetInt("RESIZE_HEIGHT"),
	}, nil
}

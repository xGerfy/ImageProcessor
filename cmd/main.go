package main

import (
	"context"
	"fmt"
	"image-processor/internal/config"
	"image-processor/internal/handler"
	"image-processor/internal/router"
	"image-processor/internal/service"
	"image-processor/internal/storage"
	"image-processor/internal/worker"
	"os"
	"os/signal"
	"syscall"

	"github.com/wb-go/wbf/zlog"
)

func main() {
	// Загружаем конфигурацию
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to load config: %v\n", err)
		os.Exit(1)
	}

	// Инициализируем логгер
	zlog.InitConsole()
	_ = zlog.SetLevel(cfg.LogLevel)

	zlog.Logger.Info().Msg("starting ImageProcessor service")

	// Создаём хранилище файлов
	fileStorage, err := storage.NewFileStorage(cfg.StorageOrig, cfg.StorageProc)
	if err != nil {
		zlog.Logger.Fatal().Err(err).Msg("failed to create file storage")
	}

	// Создаём хранилище статусов (Redis)
	var statusStorage handler.StatusStorageInterface
	statusStorage, err = storage.NewStatusStorage(cfg.RedisAddr, cfg.RedisPassword)
	if err != nil {
		zlog.Logger.Warn().Err(err).Msg("Redis not available, using in-memory storage")
		statusStorage = storage.NewInMemoryStatusStorage()
	}

	// Создаём Kafka producer/consumer
	producer := handler.NewKafkaProducer(cfg.KafkaBrokers, cfg.KafkaTopic, zlog.Logger)
	defer producer.Close()

	consumer := handler.NewKafkaConsumer(cfg.KafkaBrokers, cfg.KafkaTopic, cfg.KafkaGroupID, zlog.Logger)
	defer consumer.Close()

	// Создаём процессор изображений
	processor := service.NewImageProcessor(
		cfg.WatermarkText,
		cfg.ThumbWidth,
		cfg.ThumbHeight,
		cfg.ResizeWidth,
		cfg.ResizeHeight,
	)

	// Создаём воркер
	w := worker.NewWorker(consumer, statusStorage, processor, fileStorage, zlog.Logger)

	// Создаём обработчик HTTP
	baseURL := fmt.Sprintf("http://localhost:%d", cfg.ServerPort)
	imageHandler := handler.NewImageHandler(fileStorage, statusStorage, producer, zlog.Logger, baseURL)

	// Создаём и настраиваем роутер
	r := router.New()
	r.Setup(imageHandler, cfg.StorageProc)

	// Контекст для graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Запускаем воркер в горутине
	go w.Start(ctx)

	// Graceful shutdown
	shutdown := make(chan struct{})
	go func() {
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
		<-sigChan
		zlog.Logger.Info().Msg("shutting down...")
		cancel()
		w.Stop()
		close(shutdown)
	}()

	// Запускаем сервер
	addr := fmt.Sprintf("%s:%d", cfg.ServerAddr, cfg.ServerPort)
	zlog.Logger.Info().Str("addr", addr).Msg("starting HTTP server")

	// Запускаем сервер в горутине
	go func() {
		if err := r.Engine().Run(addr); err != nil {
			zlog.Logger.Fatal().Err(err).Msg("failed to start server")
		}
	}()

	// Ждём сигнала завершения
	<-shutdown
	zlog.Logger.Info().Msg("server stopped")
}

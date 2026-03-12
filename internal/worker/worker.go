package worker

import (
	"context"
	"image-processor/internal/handler"
	"image-processor/internal/model"
	"image-processor/internal/service"
	"image-processor/internal/storage"
	"time"

	"github.com/wb-go/wbf/zlog"
)

// StatusStorageInterface - интерфейс для хранилища статусов
type StatusStorageInterface interface {
	SaveTask(ctx context.Context, task *model.ImageTask) error
	GetTask(ctx context.Context, id string) (*model.ImageTask, error)
	UpdateStatus(ctx context.Context, id string, status model.TaskStatus, errMsg string) error
	UpdateProcessedPaths(ctx context.Context, id string, paths map[string]string) error
	DeleteTask(ctx context.Context, id string) error
}

// Worker - фоновый обработчик задач
type Worker struct {
	consumer      *handler.KafkaConsumer
	statusStorage StatusStorageInterface
	processor     *service.ImageProcessor
	fileStorage   *storage.FileStorage
	logger        zlog.Zerolog
}

// NewWorker создаёт воркер
func NewWorker(
	consumer *handler.KafkaConsumer,
	statusStorage StatusStorageInterface,
	processor *service.ImageProcessor,
	fileStorage *storage.FileStorage,
	logger zlog.Zerolog,
) *Worker {
	return &Worker{
		consumer:      consumer,
		statusStorage: statusStorage,
		processor:     processor,
		fileStorage:   fileStorage,
		logger:        logger,
	}
}

// Start запускает обработку задач
func (w *Worker) Start(ctx context.Context) {
	w.logger.Info().Msg("worker started")

	for {
		select {
		case <-ctx.Done():
			w.logger.Info().Msg("worker stopped")
			return
		default:
			w.processTask(ctx)
		}
	}
}

func (w *Worker) processTask(ctx context.Context) {
	task, err := w.consumer.FetchTask(ctx)
	if err != nil {
		w.logger.Error().Err(err).Msg("failed to fetch task")
		time.Sleep(time.Second)
		return
	}

	if task == nil {
		time.Sleep(time.Second)
		return
	}

	w.logger.Info().Str("id", task.ID).Str("path", task.OriginalPath).Msg("processing task")

	// Обновляем статус на processing
	if err := w.statusStorage.UpdateStatus(ctx, task.ID, model.StatusProcessing, ""); err != nil {
		w.logger.Error().Err(err).Str("id", task.ID).Msg("failed to update status")
		return
	}

	// Обрабатываем изображение
	processedPaths := make(map[string]string)

	// Создаём thumbnail
	thumbnailData, err := w.processor.CreateThumbnail(task.OriginalPath)
	if err != nil {
		w.logger.Error().Err(err).Str("id", task.ID).Msg("failed to create thumbnail")
	} else {
		thumbnailPath, err := w.fileStorage.SaveProcessed(task.ID, "thumbnail", thumbnailData)
		if err != nil {
			w.logger.Error().Err(err).Str("id", task.ID).Msg("failed to save thumbnail")
		} else {
			processedPaths["thumbnail"] = thumbnailPath
			w.logger.Info().Str("id", task.ID).Str("path", thumbnailPath).Msg("thumbnail created")
		}
	}

	// Создаём resize
	resizeData, err := w.processor.CreateResize(task.OriginalPath)
	if err != nil {
		w.logger.Error().Err(err).Str("id", task.ID).Msg("failed to create resize")
	} else {
		resizePath, err := w.fileStorage.SaveProcessed(task.ID, "resize", resizeData)
		if err != nil {
			w.logger.Error().Err(err).Str("id", task.ID).Msg("failed to save resize")
		} else {
			processedPaths["resize"] = resizePath
			w.logger.Info().Str("id", task.ID).Str("path", resizePath).Msg("resize created")
		}
	}

	// Создаём watermark
	watermarkData, err := w.processor.CreateWatermark(task.OriginalPath)
	if err != nil {
		w.logger.Error().Err(err).Str("id", task.ID).Msg("failed to create watermark")
	} else {
		watermarkPath, err := w.fileStorage.SaveProcessed(task.ID, "watermark", watermarkData)
		if err != nil {
			w.logger.Error().Err(err).Str("id", task.ID).Msg("failed to save watermark")
		} else {
			processedPaths["watermark"] = watermarkPath
			w.logger.Info().Str("id", task.ID).Str("path", watermarkPath).Msg("watermark created")
		}
	}

	// Обновляем статус
	if len(processedPaths) > 0 {
		if err := w.statusStorage.UpdateProcessedPaths(ctx, task.ID, processedPaths); err != nil {
			w.logger.Error().Err(err).Str("id", task.ID).Msg("failed to update processed paths")
		} else {
			w.logger.Info().Str("id", task.ID).Msg("task completed successfully")
		}
	} else {
		if err := w.statusStorage.UpdateStatus(ctx, task.ID, model.StatusFailed, "all processing failed"); err != nil {
			w.logger.Error().Err(err).Str("id", task.ID).Msg("failed to update status to failed")
		}
	}

	// Подтверждаем задачу
	if err := w.consumer.CommitTask(ctx, task); err != nil {
		w.logger.Error().Err(err).Str("id", task.ID).Msg("failed to commit task")
	}
}

// Stop останавливает воркер
func (w *Worker) Stop() {
	w.consumer.Close()
}

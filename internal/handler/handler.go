package handler

import (
	"context"
	"image-processor/internal/model"
	"image-processor/internal/storage"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/wb-go/wbf/ginext"
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

// ImageHandler - обработчик запросов изображений
type ImageHandler struct {
	fileStorage   *storage.FileStorage
	statusStorage StatusStorageInterface
	producer      *KafkaProducer
	logger        zlog.Zerolog
	baseURL       string
}

// NewImageHandler создаёт обработчик
func NewImageHandler(
	fileStorage *storage.FileStorage,
	statusStorage StatusStorageInterface,
	producer *KafkaProducer,
	logger zlog.Zerolog,
	baseURL string,
) *ImageHandler {
	return &ImageHandler{
		fileStorage:   fileStorage,
		statusStorage: statusStorage,
		producer:      producer,
		logger:        logger,
		baseURL:       baseURL,
	}
}

// Upload обрабатывает загрузку изображения
func (h *ImageHandler) Upload(c *ginext.Context) {
	file, err := c.FormFile("file")
	if err != nil {
		h.logger.Error().Err(err).Msg("failed to get file from form")
		c.JSON(http.StatusBadRequest, ginext.H{
			"error": "failed to get file: " + err.Error(),
		})
		return
	}

	// Открываем файл
	src, err := file.Open()
	if err != nil {
		h.logger.Error().Err(err).Msg("failed to open file")
		c.JSON(http.StatusInternalServerError, ginext.H{
			"error": "failed to open file: " + err.Error(),
		})
		return
	}
	defer src.Close()

	// Сохраняем оригинал
	id, originalPath, err := h.fileStorage.SaveOriginal(src, file.Filename)
	if err != nil {
		h.logger.Error().Err(err).Msg("failed to save original file")
		c.JSON(http.StatusInternalServerError, ginext.H{
			"error": "failed to save file: " + err.Error(),
		})
		return
	}

	// Создаём задачу
	task := &model.ImageTask{
		ID:           id,
		OriginalPath: originalPath,
		Filename:     file.Filename,
		ContentType:  file.Header.Get("Content-Type"),
		Status:       model.StatusPending,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	// Сохраняем статус
	if err := h.statusStorage.SaveTask(c.Request.Context(), task); err != nil {
		h.logger.Error().Err(err).Msg("failed to save task status")
		c.JSON(http.StatusInternalServerError, ginext.H{
			"error": "failed to save task: " + err.Error(),
		})
		return
	}

	// Отправляем задачу в Kafka
	if err := h.producer.SendTask(c.Request.Context(), task); err != nil {
		h.logger.Error().Err(err).Msg("failed to send task to Kafka")
		c.JSON(http.StatusInternalServerError, ginext.H{
			"error": "failed to queue task: " + err.Error(),
		})
		return
	}

	h.logger.Info().Str("id", id).Str("filename", file.Filename).Msg("image uploaded")

	c.JSON(http.StatusOK, model.UploadResponse{
		ID:       id,
		Filename: file.Filename,
		Status:   string(model.StatusPending),
		Message:  "Image uploaded successfully. Processing started.",
	})
}

// GetImage возвращает обработанное изображение
func (h *ImageHandler) GetImage(c *ginext.Context) {
	id := c.Param("id")
	processType := c.Query("type") // thumbnail, resize, watermark

	task, err := h.statusStorage.GetTask(c.Request.Context(), id)
	if err != nil {
		h.logger.Error().Err(err).Str("id", id).Msg("failed to get task")
		c.JSON(http.StatusInternalServerError, ginext.H{
			"error": "failed to get image: " + err.Error(),
		})
		return
	}

	if task == nil {
		c.JSON(http.StatusNotFound, ginext.H{
			"error": "image not found",
		})
		return
	}

	if task.Status == model.StatusFailed {
		c.JSON(http.StatusInternalServerError, ginext.H{
			"error": "processing failed: " + task.Error,
		})
		return
	}

	if task.Status == model.StatusPending || task.Status == model.StatusProcessing {
		c.JSON(http.StatusOK, ginext.H{
			"status":  "processing",
			"message": "Image is still being processed",
		})
		return
	}

	// Определяем какой файл отдать
	var filePath string
	if processType != "" {
		filePath = task.ProcessedPaths[processType]
	} else {
		// По умолчанию отдаём thumbnail
		filePath = task.ProcessedPaths["thumbnail"]
	}

	if filePath == "" {
		filePath = task.OriginalPath
	}

	// Проверяем существование файла
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		c.JSON(http.StatusNotFound, ginext.H{
			"error": "file not found",
		})
		return
	}

	h.logger.Info().Str("id", id).Str("path", filePath).Msg("serving image")

	// Определяем Content-Type по расширению
	contentType := "image/jpeg"
	if filepath.Ext(filePath) == ".png" {
		contentType = "image/png"
	} else if filepath.Ext(filePath) == ".gif" {
		contentType = "image/gif"
	}
	c.Header("Content-Type", contentType)
	c.File(filePath)
}

// DeleteImage удаляет изображение
func (h *ImageHandler) DeleteImage(c *ginext.Context) {
	id := c.Param("id")

	task, err := h.statusStorage.GetTask(c.Request.Context(), id)
	if err != nil {
		h.logger.Error().Err(err).Str("id", id).Msg("failed to get task")
		c.JSON(http.StatusInternalServerError, ginext.H{
			"error": "failed to delete image: " + err.Error(),
		})
		return
	}

	if task == nil {
		c.JSON(http.StatusNotFound, ginext.H{
			"error": "image not found",
		})
		return
	}

	// Удаляем файлы
	if err := h.fileStorage.DeleteOriginal(id); err != nil {
		h.logger.Error().Err(err).Str("id", id).Msg("failed to delete original")
	}
	if err := h.fileStorage.DeleteProcessed(id); err != nil {
		h.logger.Error().Err(err).Str("id", id).Msg("failed to delete processed")
	}

	// Удаляем задачу
	if err := h.statusStorage.DeleteTask(c.Request.Context(), id); err != nil {
		h.logger.Error().Err(err).Str("id", id).Msg("failed to delete task")
	}

	h.logger.Info().Str("id", id).Msg("image deleted")

	c.JSON(http.StatusOK, ginext.H{
		"message": "Image deleted successfully",
	})
}

// GetStatus возвращает статус обработки
func (h *ImageHandler) GetStatus(c *ginext.Context) {
	id := c.Param("id")

	task, err := h.statusStorage.GetTask(c.Request.Context(), id)
	if err != nil {
		h.logger.Error().Err(err).Str("id", id).Msg("failed to get task")
		c.JSON(http.StatusInternalServerError, ginext.H{
			"error": "failed to get status: " + err.Error(),
		})
		return
	}

	if task == nil {
		c.JSON(http.StatusNotFound, ginext.H{
			"error": "image not found",
		})
		return
	}

	// Формируем URLs
	processedURLs := make(map[string]string)
	for ptype := range task.ProcessedPaths {
		processedURLs[ptype] = h.baseURL + "/image/" + id + "?type=" + ptype
	}

	response := model.ImageInfo{
		ID:            task.ID,
		Filename:      task.Filename,
		Status:        task.Status,
		OriginalURL:   h.baseURL + "/image/" + id + "?type=original",
		ProcessedURLs: processedURLs,
		Error:         task.Error,
		CreatedAt:     task.CreatedAt,
	}

	c.JSON(http.StatusOK, response)
}

// ListImages возвращает список всех изображений
func (h *ImageHandler) ListImages(c *ginext.Context) {
	// В реальной реализации нужно хранить список всех ID
	// Для простоты возвращаем пустой список
	c.JSON(http.StatusOK, ginext.H{
		"images": []model.ImageInfo{},
	})
}

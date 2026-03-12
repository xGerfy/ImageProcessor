package model

import "time"

// TaskStatus - статус обработки изображения
type TaskStatus string

const (
	StatusPending    TaskStatus = "pending"
	StatusProcessing TaskStatus = "processing"
	StatusCompleted  TaskStatus = "completed"
	StatusFailed     TaskStatus = "failed"
)

// ImageTask - задача на обработку изображения
type ImageTask struct {
	ID             string            `json:"id"`
	OriginalPath   string            `json:"original_path"`
	Filename       string            `json:"filename"`
	ContentType    string            `json:"content_type"`
	Status         TaskStatus        `json:"status"`
	ProcessedPaths map[string]string `json:"processed_paths,omitempty"` // type -> path
	Error          string            `json:"error,omitempty"`
	CreatedAt      time.Time         `json:"created_at"`
	UpdatedAt      time.Time         `json:"updated_at"`
}

// ProcessedImage - обработанное изображение
type ProcessedImage struct {
	Type     string `json:"type"` // thumbnail, resize, watermark
	Path     string `json:"path"`
	Width    int    `json:"width"`
	Height   int    `json:"height"`
	FileSize int64  `json:"file_size"`
}

// UploadResponse - ответ на загрузку изображения
type UploadResponse struct {
	ID       string `json:"id"`
	Filename string `json:"filename"`
	Status   string `json:"status"`
	Message  string `json:"message"`
}

// ImageInfo - информация об изображении для API
type ImageInfo struct {
	ID            string            `json:"id"`
	Filename      string            `json:"filename"`
	Status        TaskStatus        `json:"status"`
	OriginalURL   string            `json:"original_url,omitempty"`
	ProcessedURLs map[string]string `json:"processed_urls,omitempty"`
	Error         string            `json:"error,omitempty"`
	CreatedAt     time.Time         `json:"created_at"`
}

package storage

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/google/uuid"
)

// FileStorage - хранилище файлов
type FileStorage struct {
	originalsPath string
	processedPath string
}

// NewFileStorage создаёт новое хранилище
func NewFileStorage(originalsPath, processedPath string) (*FileStorage, error) {
	if err := os.MkdirAll(originalsPath, 0755); err != nil {
		return nil, fmt.Errorf("failed to create originals directory: %w", err)
	}
	if err := os.MkdirAll(processedPath, 0755); err != nil {
		return nil, fmt.Errorf("failed to create processed directory: %w", err)
	}

	return &FileStorage{
		originalsPath: originalsPath,
		processedPath: processedPath,
	}, nil
}

// SaveOriginal сохраняет оригинальный файл и возвращает его ID и путь
func (s *FileStorage) SaveOriginal(file io.Reader, filename string) (string, string, error) {
	id := uuid.New().String()
	ext := filepath.Ext(filename)
	if ext == "" {
		ext = ".jpg"
	}

	newFilename := id + ext
	fullPath := filepath.Join(s.originalsPath, newFilename)

	outFile, err := os.Create(fullPath)
	if err != nil {
		return "", "", fmt.Errorf("failed to create file: %w", err)
	}
	defer outFile.Close()

	_, err = io.Copy(outFile, file)
	if err != nil {
		os.Remove(fullPath)
		return "", "", fmt.Errorf("failed to save file: %w", err)
	}

	return id, fullPath, nil
}

// SaveProcessed сохраняет обработанный файл
func (s *FileStorage) SaveProcessed(id string, processType string, data []byte) (string, error) {
	// Определяем формат по первым байтам
	ext := getFileExtension(data)
	filename := fmt.Sprintf("%s_%s%s", id, processType, ext)
	fullPath := filepath.Join(s.processedPath, filename)

	err := os.WriteFile(fullPath, data, 0644)
	if err != nil {
		return "", fmt.Errorf("failed to save processed file: %w", err)
	}

	return fullPath, nil
}

// getFileExtension определяет расширение файла по заголовку
func getFileExtension(data []byte) string {
	if len(data) < 4 {
		return ".jpg"
	}

	// PNG: 89 50 4E 47
	if data[0] == 0x89 && data[1] == 0x50 && data[2] == 0x4E && data[3] == 0x47 {
		return ".png"
	}

	// GIF: 47 49 46 38
	if data[0] == 0x47 && data[1] == 0x49 && data[2] == 0x46 && data[3] == 0x38 {
		return ".gif"
	}

	// JPEG: FF D8 FF
	if data[0] == 0xFF && data[1] == 0xD8 && data[2] == 0xFF {
		return ".jpg"
	}

	// По умолчанию JPEG
	return ".jpg"
}

// GetOriginal возвращает путь к оригинальному файлу
func (s *FileStorage) GetOriginal(id string) (string, error) {
	pattern := filepath.Join(s.originalsPath, id+".*")
	matches, err := filepath.Glob(pattern)
	if err != nil {
		return "", err
	}
	if len(matches) == 0 {
		return "", os.ErrNotExist
	}
	return matches[0], nil
}

// GetProcessed возвращает путь к обработанному файлу
func (s *FileStorage) GetProcessed(id string, processType string) (string, error) {
	pattern := filepath.Join(s.processedPath, fmt.Sprintf("%s_%s.*", id, processType))
	matches, err := filepath.Glob(pattern)
	if err != nil {
		return "", err
	}
	if len(matches) == 0 {
		return "", os.ErrNotExist
	}
	return matches[0], nil
}

// DeleteOriginal удаляет оригинальный файл
func (s *FileStorage) DeleteOriginal(id string) error {
	pattern := filepath.Join(s.originalsPath, id+".*")
	matches, err := filepath.Glob(pattern)
	if err != nil {
		return err
	}
	for _, match := range matches {
		if err := os.Remove(match); err != nil {
			return err
		}
	}
	return nil
}

// DeleteProcessed удаляет все обработанные файлы для данного ID
func (s *FileStorage) DeleteProcessed(id string) error {
	pattern := filepath.Join(s.processedPath, fmt.Sprintf("%s_*.*", id))
	matches, err := filepath.Glob(pattern)
	if err != nil {
		return err
	}
	for _, match := range matches {
		if err := os.Remove(match); err != nil {
			return err
		}
	}
	return nil
}

// GetOriginalsPath возвращает путь к директории оригиналов
func (s *FileStorage) GetOriginalsPath() string {
	return s.originalsPath
}

// GetProcessedPath возвращает путь к директории обработанных файлов
func (s *FileStorage) GetProcessedPath() string {
	return s.processedPath
}

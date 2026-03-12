package storage

import (
	"context"
	"image-processor/internal/model"
	"sync"
	"time"
)

// InMemoryStatusStorage - in-memory хранилище статусов
type InMemoryStatusStorage struct {
	mu   sync.RWMutex
	data map[string]*model.ImageTask
}

// NewInMemoryStatusStorage создаёт in-memory хранилище
func NewInMemoryStatusStorage() *InMemoryStatusStorage {
	return &InMemoryStatusStorage{
		data: make(map[string]*model.ImageTask),
	}
}

// SaveTask сохраняет задачу
func (s *InMemoryStatusStorage) SaveTask(ctx context.Context, task *model.ImageTask) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Копируем задачу
	taskCopy := *task
	s.data[task.ID] = &taskCopy
	return nil
}

// GetTask получает задачу
func (s *InMemoryStatusStorage) GetTask(ctx context.Context, id string) (*model.ImageTask, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	task, ok := s.data[id]
	if !ok {
		return nil, nil
	}

	taskCopy := *task
	return &taskCopy, nil
}

// UpdateStatus обновляет статус задачи
func (s *InMemoryStatusStorage) UpdateStatus(ctx context.Context, id string, status model.TaskStatus, errMsg string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	task, ok := s.data[id]
	if !ok {
		return nil
	}

	task.Status = status
	task.Error = errMsg
	task.UpdatedAt = time.Now()
	return nil
}

// UpdateProcessedPaths обновляет пути к обработанным файлам
func (s *InMemoryStatusStorage) UpdateProcessedPaths(ctx context.Context, id string, paths map[string]string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	task, ok := s.data[id]
	if !ok {
		return nil
	}

	task.ProcessedPaths = paths
	task.Status = model.StatusCompleted
	task.UpdatedAt = time.Now()
	return nil
}

// DeleteTask удаляет задачу
func (s *InMemoryStatusStorage) DeleteTask(ctx context.Context, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	delete(s.data, id)
	return nil
}

// Close закрывает хранилище (noop для in-memory)
func (s *InMemoryStatusStorage) Close() error {
	return nil
}

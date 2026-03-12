package storage

import (
	"context"
	"encoding/json"
	"image-processor/internal/model"
	"time"

	"github.com/wb-go/wbf/redis"
)

// StatusStorage - хранилище статусов в Redis
type StatusStorage struct {
	client *redis.Client
}

// NewStatusStorage создаёт новое хранилище статусов
func NewStatusStorage(addr, password string) (*StatusStorage, error) {
	client := redis.New(addr, password, 0)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx); err != nil {
		return nil, err
	}

	return &StatusStorage{client: client}, nil
}

// SaveTask сохраняет задачу
func (s *StatusStorage) SaveTask(ctx context.Context, task *model.ImageTask) error {
	data, err := json.Marshal(task)
	if err != nil {
		return err
	}

	key := s.taskKey(task.ID)
	return s.client.Set(ctx, key, string(data))
}

// GetTask получает задачу
func (s *StatusStorage) GetTask(ctx context.Context, id string) (*model.ImageTask, error) {
	key := s.taskKey(id)
	value, err := s.client.Get(ctx, key)
	if err != nil {
		if err == redis.NoMatches {
			return nil, nil
		}
		return nil, err
	}

	var task model.ImageTask
	if err := json.Unmarshal([]byte(value), &task); err != nil {
		return nil, err
	}

	return &task, nil
}

// UpdateStatus обновляет статус задачи
func (s *StatusStorage) UpdateStatus(ctx context.Context, id string, status model.TaskStatus, errMsg string) error {
	task, err := s.GetTask(ctx, id)
	if err != nil {
		return err
	}
	if task == nil {
		return nil
	}

	task.Status = status
	task.Error = errMsg
	task.UpdatedAt = time.Now()

	return s.SaveTask(ctx, task)
}

// UpdateProcessedPaths обновляет пути к обработанным файлам
func (s *StatusStorage) UpdateProcessedPaths(ctx context.Context, id string, paths map[string]string) error {
	task, err := s.GetTask(ctx, id)
	if err != nil {
		return err
	}
	if task == nil {
		return nil
	}

	task.ProcessedPaths = paths
	task.Status = model.StatusCompleted
	task.UpdatedAt = time.Now()

	return s.SaveTask(ctx, task)
}

// DeleteTask удаляет задачу
func (s *StatusStorage) DeleteTask(ctx context.Context, id string) error {
	key := s.taskKey(id)
	return s.client.Del(ctx, key)
}

func (s *StatusStorage) taskKey(id string) string {
	return "image_task:" + id
}

// Close закрывает соединение
func (s *StatusStorage) Close() error {
	return nil
}

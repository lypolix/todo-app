package service

import (
	"context"
	"fmt"
	"time"

	"github.com/lypolix/todo-app"
	"github.com/lypolix/todo-app/pkg/cache"
	"github.com/lypolix/todo-app/pkg/repository"
	"github.com/lypolix/todo-app/pkg/websocket"
	"github.com/sirupsen/logrus"
)

type TodoItemService struct {
	repo     repository.TodoItem
	listRepo repository.TodoList
	cache    *cache.CacheService
	wsHub    *websocket.Hub
}

func NewTodoItemService(repo repository.TodoItem, listRepo repository.TodoList, 
					   cache *cache.CacheService, wsHub *websocket.Hub) *TodoItemService {
	return &TodoItemService{
		repo:     repo,
		listRepo: listRepo,
		cache:    cache,
		wsHub:    wsHub,
	}
}

func (s *TodoItemService) Create(userId, listId int, item todo.TodoItem) (int, error) {
	_, err := s.listRepo.GetById(userId, listId)
	if err != nil {
		return 0, err
	}

	id, err := s.repo.Create(userId, listId, item)
	if err != nil {
		return 0, err
	}

	// Очистка кэша
	cacheKey := fmt.Sprintf("items:%d:%d", userId, listId)
	s.cache.Delete(context.Background(), cacheKey)

	logrus.WithFields(logrus.Fields{
		"user_id": userId,
		"list_id": listId,
		"item_id": id,
		"title":   item.Title,
	}).Info("Todo item created")

	return id, nil
}

func (s *TodoItemService) GetAll(userId, listId int) ([]todo.TodoItem, error) {
	cacheKey := fmt.Sprintf("items:%d:%d", userId, listId)
	
	var items []todo.TodoItem
	if err := s.cache.Get(context.Background(), cacheKey, &items); err == nil {
		logrus.WithField("cache_key", cacheKey).Debug("Cache hit for todo items")
		return items, nil
	}

	items, err := s.repo.GetAll(userId, listId)
	if err != nil {
		return nil, err
	}

	// Кэширование на 5 минут
	s.cache.Set(context.Background(), cacheKey, items, 5*time.Minute)
	
	return items, nil
}

func (s *TodoItemService) GetById(userId, itemId int) (todo.TodoItem, error) {
	cacheKey := fmt.Sprintf("item:%d:%d", userId, itemId)
	
	var item todo.TodoItem
	if err := s.cache.Get(context.Background(), cacheKey, &item); err == nil {
		logrus.WithField("cache_key", cacheKey).Debug("Cache hit for todo item")
		return item, nil
	}

	item, err := s.repo.GetById(userId, itemId)
	if err != nil {
		return item, err
	}

	// Кэширование на 5 минут
	s.cache.Set(context.Background(), cacheKey, item, 5*time.Minute)
	
	return item, nil
}

func (s *TodoItemService) Update(userId, itemId int, input todo.UpdateItemInput) error {
	err := s.repo.Update(userId, itemId, input)
	if err != nil {
		return err
	}

	// Очистка кэша
	s.cache.Delete(context.Background(), fmt.Sprintf("item:%d:%d", userId, itemId))
	s.cache.DeletePattern(context.Background(), fmt.Sprintf("items:%d:*", userId))

	logrus.WithFields(logrus.Fields{
		"user_id": userId,
		"item_id": itemId,
		"action":  "updated",
	}).Info("Todo item updated")

	return nil
}

func (s *TodoItemService) Delete(userId, itemId int) error {
	err := s.repo.Delete(userId, itemId)
	if err != nil {
		return err
	}

	// Очистка кэша
	s.cache.Delete(context.Background(), fmt.Sprintf("item:%d:%d", userId, itemId))
	s.cache.DeletePattern(context.Background(), fmt.Sprintf("items:%d:*", userId))

	logrus.WithFields(logrus.Fields{
		"user_id": userId,
		"item_id": itemId,
		"action":  "deleted",
	}).Info("Todo item deleted")

	return nil
}

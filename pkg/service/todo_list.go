package service

import (
	"context"
	"fmt"
	"time"

	"github.com/lypolix/todo-app"
	"github.com/lypolix/todo-app/pkg/cache"
	"github.com/lypolix/todo-app/pkg/repository"
	"github.com/sirupsen/logrus"
)

type TodoListService struct {
	repo  repository.TodoList
	cache *cache.CacheService
}

func NewTodoListService(repo repository.TodoList, cache *cache.CacheService) *TodoListService {
	return &TodoListService{
		repo:  repo,
		cache: cache,
	}
}

func (s *TodoListService) Create(userId int, list todo.TodoList) (int, error) {
	id, err := s.repo.Create(userId, list)
	if err != nil {
		return 0, err
	}

	
	cacheKey := fmt.Sprintf("lists:%d", userId)
	s.cache.Delete(context.Background(), cacheKey)

	logrus.WithFields(logrus.Fields{
		"user_id": userId,
		"list_id": id,
		"title":   list.Title,
	}).Info("Todo list created")

	return id, nil
}

func (s *TodoListService) GetAll(userId int) ([]todo.TodoList, error) {
	cacheKey := fmt.Sprintf("lists:%d", userId)
	
	var lists []todo.TodoList
	if err := s.cache.Get(context.Background(), cacheKey, &lists); err == nil {
		logrus.WithField("cache_key", cacheKey).Debug("Cache hit for todo lists")
		return lists, nil
	}

	lists, err := s.repo.GetAll(userId)
	if err != nil {
		return nil, err
	}

	
	s.cache.Set(context.Background(), cacheKey, lists, 5*time.Minute)
	
	return lists, nil
}

func (s *TodoListService) GetById(userId, listId int) (todo.TodoList, error) {
	cacheKey := fmt.Sprintf("list:%d:%d", userId, listId)
	
	var list todo.TodoList
	if err := s.cache.Get(context.Background(), cacheKey, &list); err == nil {
		logrus.WithField("cache_key", cacheKey).Debug("Cache hit for todo list")
		return list, nil
	}

	list, err := s.repo.GetById(userId, listId)
	if err != nil {
		return list, err
	}

	
	s.cache.Set(context.Background(), cacheKey, list, 5*time.Minute)
	
	return list, nil
}

func (s *TodoListService) Delete(userId, listId int) error {
	err := s.repo.Delete(userId, listId)
	if err != nil {
		return err
	}

	
	s.cache.Delete(context.Background(), fmt.Sprintf("list:%d:%d", userId, listId))
	s.cache.Delete(context.Background(), fmt.Sprintf("lists:%d", userId))

	logrus.WithFields(logrus.Fields{
		"user_id": userId,
		"list_id": listId,
		"action":  "deleted",
	}).Info("Todo list deleted")

	return nil
}

func (s *TodoListService) Update(userId, listId int, input todo.UpdateListInput) error {
	if err := input.Validate(); err != nil {
		return err
	}

	err := s.repo.Update(userId, listId, input)
	if err != nil {
		return err
	}

	
	s.cache.Delete(context.Background(), fmt.Sprintf("list:%d:%d", userId, listId))
	s.cache.Delete(context.Background(), fmt.Sprintf("lists:%d", userId))

	logrus.WithFields(logrus.Fields{
		"user_id": userId,
		"list_id": listId,
		"action":  "updated",
	}).Info("Todo list updated")

	return nil
}

package infrastructure

import (
	"github.com/swaggest/rest/_examples/task-api/internal/infrastructure/repository"
	"github.com/swaggest/rest/_examples/task-api/internal/infrastructure/service"
)

func NewServiceLocator() (*service.Locator, error) {
	l := service.Locator{}

	taskRepository := repository.Task{}

	l.TaskCloserProvider = &taskRepository
	l.TaskFinderProvider = &taskRepository
	l.TaskUpdaterProvider = &taskRepository
	l.TaskCreatorProvider = &taskRepository

	return &l, nil
}

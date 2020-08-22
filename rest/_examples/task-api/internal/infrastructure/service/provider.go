package service

import "github.com/swaggest/rest/_examples/task-api/internal/domain/task"

type TaskCreatorProvider interface {
	TaskCreator() task.Creator
}

type TaskUpdaterProvider interface {
	TaskUpdater() task.Updater
}

type TaskFinderProvider interface {
	TaskFinder() task.Finder
}

type TaskCloserProvider interface {
	TaskCloser() task.Closer
}

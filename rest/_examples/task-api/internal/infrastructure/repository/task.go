package repository

import (
	"context"
	"errors"
	"sort"
	"sync"
	"time"

	"github.com/swaggest/rest/_examples/task-api/internal/domain/task"
	"github.com/swaggest/usecase"
	"github.com/swaggest/usecase/status"
)

type Task struct {
	mu     sync.Mutex
	lastID int
	list   map[task.Identity]task.Entity
}

func (tr *Task) TaskUpdater() task.Updater {
	return tr
}

func (tr *Task) Update(ctx context.Context, identity task.Identity, value task.Value) error {
	tr.mu.Lock()
	defer tr.mu.Unlock()

	t, found := tr.list[identity]
	if !found {
		return status.NotFound
	}

	if t.ClosedAt != nil {
		return status.Wrap(errors.New("task is already closed"), status.FailedPrecondition)
	}

	t.Value = value
	tr.list[identity] = t

	return nil
}

func (tr *Task) TaskFinder() task.Finder {
	return tr
}

func (tr *Task) Find(ctx context.Context) []task.Entity {
	tr.mu.Lock()
	defer tr.mu.Unlock()

	result := make([]task.Entity, 0, len(tr.list))
	for _, t := range tr.list {
		result = append(result, t)
	}

	sort.Slice(result, func(i, j int) bool {
		return result[i].ID < result[j].ID
	})

	return result
}

func (tr *Task) FindByID(ctx context.Context, identity task.Identity) (task.Entity, error) {
	tr.mu.Lock()
	defer tr.mu.Unlock()

	t, found := tr.list[identity]
	if !found {
		return task.Entity{}, status.NotFound
	}

	return t, nil
}

func (tr *Task) TaskCloser() task.Closer {
	return tr
}

func (tr *Task) Close(ctx context.Context, identity task.Identity) error {
	return tr.close(identity, task.Done)
}

func (tr *Task) Cancel(ctx context.Context, identity task.Identity) error {
	return tr.close(identity, task.Canceled)
}

func (tr *Task) close(identity task.Identity, st task.Status) error {
	tr.mu.Lock()
	defer tr.mu.Unlock()

	t, found := tr.list[identity]
	if !found {
		return status.NotFound
	}

	if t.ClosedAt != nil {
		return status.Wrap(errors.New("task is already closed"), status.FailedPrecondition)
	}

	now := time.Now()
	t.ClosedAt = &now
	t.Status = st
	tr.list[t.Identity] = t

	return nil

}

func (tr *Task) TaskCreator() task.Creator {
	return tr
}

func (tr *Task) Create(ctx context.Context, value task.Value) (task.Entity, error) {
	tr.mu.Lock()
	defer tr.mu.Unlock()

	for _, t := range tr.list {
		if t.Value.Goal == value.Goal {
			return task.Entity{}, usecase.Error{
				StatusCode: status.AlreadyExists,
				Context: map[string]interface{}{
					"task": t,
				},
				Value: errors.New("task with same goal already exists"),
			}
		}
	}

	tr.lastID++

	if tr.list == nil {
		tr.list = make(map[task.Identity]task.Entity, 1)
	}

	t := task.Entity{}
	t.Value = value
	t.ID = tr.lastID
	t.CreatedAt = time.Now()
	tr.list[t.Identity] = t

	return t, nil
}

func (tr *Task) CloseExpired(ctx context.Context) error {
	tr.mu.Lock()
	defer tr.mu.Unlock()

	now := time.Now()
	for _, t := range tr.list {
		if t.Deadline != nil && now.After(*t.Deadline) {
			err := tr.close(t.Identity, task.Expired)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

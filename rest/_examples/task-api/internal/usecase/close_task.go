package usecase

import (
	"context"
	"github.com/swaggest/usecase/status"

	"github.com/swaggest/rest/_examples/task-api/internal/domain/task"
	"github.com/swaggest/usecase"
)

func CloseTask(deps interface {
	TaskCloser() task.Closer
}) usecase.Interactor {
	u := struct {
		usecase.Interactor
		usecase.Info
		usecase.WithInput
		usecase.WithOutput
	}{}

	u.SetTitle("Close Task")
	u.SetDescription("Close task by ID.")
	u.Input = new(task.Identity)
	u.SetExpectedErrors(
		status.NotFound,
		status.InvalidArgument,
	)
	u.SetTags("Tasks")

	u.Interactor = usecase.Interact(func(ctx context.Context, input, _ interface{}) error {
		var (
			in  = input.(*task.Identity)
			err error
		)

		err = deps.TaskCloser().Close(ctx, *in)

		return err
	})

	return u
}

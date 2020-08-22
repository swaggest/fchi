package usecase

import (
	"context"
	"github.com/swaggest/usecase/status"

	"github.com/swaggest/rest/_examples/task-api/internal/domain/task"
	"github.com/swaggest/usecase"
)

func CreateTask(deps interface {
	TaskCreator() task.Creator
}) usecase.Interactor {
	u := struct {
		usecase.Interactor
		usecase.Info
		usecase.WithInput
		usecase.WithOutput
	}{}

	u.SetTitle("Create Task")
	u.SetDescription("Create task to be done.")
	u.Input = new(task.Value)
	u.Output = new(task.Entity)
	u.SetExpectedErrors(
		status.AlreadyExists,
		status.InvalidArgument,
	)
	u.SetTags("Tasks")

	u.Interactor = usecase.Interact(func(ctx context.Context, input, output interface{}) error {
		var (
			in  = input.(*task.Value)
			out = output.(*task.Entity)
			err error
		)

		*out, err = deps.TaskCreator().Create(ctx, *in)

		return err
	})

	return u
}

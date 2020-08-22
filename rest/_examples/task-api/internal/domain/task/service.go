package task

import "context"

type Creator interface {
	Create(context.Context, Value) (Entity, error)
}

type Updater interface {
	Update(context.Context, Identity, Value) error
}

type Closer interface {
	Cancel(context.Context, Identity) error
	Close(context.Context, Identity) error
}

type Finder interface {
	Find(context.Context) []Entity
	FindByID(context.Context, Identity) (Entity, error)
}

package task

import (
	"encoding/json"
	"time"

	"github.com/swaggest/jsonschema-go"
)

type Status string

const (
	Active   = Status("")
	Canceled = Status("canceled")
	Done     = Status("done")
	Expired  = Status("expired")
)

func (Status) JSONSchema() ([]byte, error) {
	s := jsonschema.Schema{}
	s.
		WithType(jsonschema.String.Type()).
		WithTitle("Goal Status").
		WithDescription("Non-empty task status indicates result.").
		WithEnum(Active, Canceled, Done, Expired)

	return json.Marshal(s)
}

type Identity struct {
	ID int `json:"id" path:"id"`
}

type Value struct {
	Goal     string     `json:"goal" minLength:"1" required:"true"`
	Deadline *time.Time `json:"deadline,omitempty"`
}

type Entity struct {
	Identity
	Value
	CreatedAt time.Time  `json:"createdAt"`
	Status    Status     `json:"status,omitempty"`
	ClosedAt  *time.Time `json:"closedAt,omitempty"`
}

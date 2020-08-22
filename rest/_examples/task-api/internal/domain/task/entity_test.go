package task

import "testing"

func TestTodoStatus_JSONSchema(t *testing.T) {
	j, _ := Status("").JSONSchema()
	println(string(j))
}

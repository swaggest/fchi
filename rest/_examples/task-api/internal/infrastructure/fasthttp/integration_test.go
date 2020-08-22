package fasthttp_test

import (
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/swaggest/fchi"
	"github.com/swaggest/rest/_examples/task-api/internal/infrastructure"
	http3 "github.com/swaggest/rest/_examples/task-api/internal/infrastructure/fasthttp"
	"github.com/swaggest/rest/resttest"
	"github.com/valyala/fasthttp"
	"net/http"
	"testing"
)

func Test_taskLifeSpan(t *testing.T) {
	l, err := infrastructure.NewServiceLocator()
	require.NoError(t, err)

	r := http3.NewRouter(l)

	go fasthttp.ListenAndServe(":8085", fchi.RequestHandler(r))
	rc := resttest.NewClient("http://localhost:8085/")

	rc.WithMethod(http.MethodPost).WithPath("/v0/tasks").
		WithBody(`{"deadline": "2020-05-17T11:12:42.085Z","goal": "string"}`).
		Concurrently()

	assert.NoError(t, rc.ExpectResponseStatus(http.StatusOK))
	assert.NoError(t, rc.ExpectResponseBody([]byte(`{"createdAt": "<ignore-diff>","deadline": "2020-05-17T11:12:42.085Z","goal": "string","id": 1}`)))

	assert.NoError(t, rc.ExpectOtherResponsesStatus(http.StatusConflict))
	assert.NoError(t, rc.ExpectOtherResponsesBody([]byte(`{"status":"ALREADY_EXISTS","error":"already exists: task with same goal already exists","context":{"task":{"id":1,"goal":"string","deadline":"2020-05-17T11:12:42.085Z","createdAt":"<ignore-diff>"}}}`)))

	rc.Reset().WithMethod(http.MethodPost).WithPath("/v0/tasks").
		WithBody(`{"deadline": "2020-35-17T11:12:42.085Z","goal": "string"}`).
		Concurrently()

	assert.NoError(t, rc.ExpectResponseStatus(http.StatusBadRequest))
	assert.NoError(t, rc.ExpectResponseBody([]byte(`{"status":"INVALID_ARGUMENT","error":"invalid argument: validation failed","context":{"validationErrors":{"body":["#/deadline: \"2020-35-17T11:12:42.085Z\" is not valid \"date-time\""]}}}`)))

	rc.Reset().WithMethod(http.MethodPost).WithPath("/v0/tasks").
		WithBody(`{"deadline": "2020-05-17T11:12:42.085Z","goal": ""}`).
		Concurrently()

	assert.NoError(t, rc.ExpectResponseStatus(http.StatusBadRequest))
	assert.NoError(t, rc.ExpectResponseBody([]byte(`{"status":"INVALID_ARGUMENT","error":"invalid argument: validation failed","context":{"validationErrors":{"body":["#/goal: length must be \u003e= 1, but got 0"]}}}`)))

	rc.Reset().WithMethod(http.MethodGet).WithPath("/v0/tasks/1").Concurrently()
	assert.NoError(t, rc.ExpectResponseStatus(http.StatusOK))
	assert.NoError(t, rc.ExpectResponseBody([]byte(`{"id":1,"goal":"string","deadline":"2020-05-17T11:12:42.085Z","createdAt":"<ignore-diff>"}`)))

	rc.Reset().WithMethod(http.MethodGet).WithPath("/v0/tasks").Concurrently()
	assert.NoError(t, rc.ExpectResponseStatus(http.StatusOK))
	assert.NoError(t, rc.ExpectResponseBody([]byte(`[{"id":1,"goal":"string","deadline":"2020-05-17T11:12:42.085Z","createdAt":"<ignore-diff>"}]`)))

	rc.Reset().WithMethod(http.MethodPut).WithPath("/v0/tasks/1").
		WithBody(`{"deadline": "2020-05-17T11:12:42.085Z","goal": "foo"}`).
		Concurrently()
	assert.NoError(t, rc.ExpectResponseStatus(http.StatusNoContent))
	assert.NoError(t, rc.ExpectResponseBody(nil))

	rc.Reset().WithMethod(http.MethodGet).WithPath("/v0/tasks/1").Concurrently()
	assert.NoError(t, rc.ExpectResponseStatus(http.StatusOK))
	assert.NoError(t, rc.ExpectResponseBody([]byte(`{"id":1,"goal":"foo","deadline":"2020-05-17T11:12:42.085Z","createdAt":"<ignore-diff>"}`)))

	rc.Reset().WithMethod(http.MethodDelete).WithPath("/v0/tasks/1").Concurrently()
	assert.NoError(t, rc.ExpectResponseStatus(http.StatusNoContent))
	assert.NoError(t, rc.ExpectResponseBody(nil))

	assert.NoError(t, rc.ExpectOtherResponsesStatus(http.StatusBadRequest))
	assert.NoError(t, rc.ExpectOtherResponsesBody([]byte(`{"status":"FAILED_PRECONDITION","error":"failed precondition: task is already closed"}`)))
}

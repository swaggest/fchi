package http

import (
	"context"
	"fmt"
	"github.com/stretchr/testify/assert"
	"github.com/swaggest/rest/_examples/task-api/internal/domain/task"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/swaggest/rest/_examples/task-api/internal/infrastructure"
	"github.com/valyala/fasthttp"
)

func Benchmark_notFoundSrvFast(b *testing.B) {
	l, err := infrastructure.NewServiceLocator()
	require.NoError(b, err)

	r := NewRouter(l)
	srv := httptest.NewServer(r)

	concurrentBench(b, 50, func(i int) {
		req := fasthttp.AcquireRequest()
		resp := fasthttp.AcquireResponse()
		defer func() {
			fasthttp.ReleaseRequest(req)
			fasthttp.ReleaseResponse(resp)
		}()

		req.SetRequestURI(srv.URL + "/v0/tasks/1")

		err := fasthttp.Do(req, resp)
		if err != nil {
			b.Fatal(err.Error())
		}

		if resp.StatusCode() != http.StatusNotFound {
			failIteration(i, resp.StatusCode(), resp.Body())
		}
	})
}

//Benchmark_invalidBody-16    	  102820	     11532 ns/op	     86676 RPS	   13751 B/op	     139 allocs/op
//Benchmark_invalidBody-16    	  109832	     11131 ns/op	     89832 RPS	   12944 B/op	     128 allocs/op
//Benchmark_invalidBody-16    	  106430	     10562 ns/op	     94667 RPS	   13053 B/op	     130 allocs/op
//Benchmark_invalidBody-16    	  105198	     10640 ns/op	     93981 RPS	   12963 B/op	     128 allocs/op
func Benchmark_invalidBody(b *testing.B) {
	l, err := infrastructure.NewServiceLocator()
	require.NoError(b, err)

	r := NewRouter(l)
	srv := httptest.NewServer(r)

	tt, err := l.TaskCreator().Create(context.Background(), task.Value{Goal: "win"})
	require.NoError(b, err)
	assert.Equal(b, 1, tt.ID)

	body := []byte(`{"goal":""}`)

	concurrentBench(b, 50, func(i int) {
		req := fasthttp.AcquireRequest()
		resp := fasthttp.AcquireResponse()
		defer func() {
			fasthttp.ReleaseRequest(req)
			fasthttp.ReleaseResponse(resp)
		}()

		req.Header.SetMethod(http.MethodPut)
		req.Header.SetContentType("application/json")
		req.SetRequestURI(srv.URL + "/v0/tasks/1")
		req.SetBody(body)

		err := fasthttp.Do(req, resp)
		if err != nil {
			b.Fatal(err.Error())
		}

		if resp.StatusCode() != http.StatusBadRequest {
			failIteration(i, resp.StatusCode(), resp.Body())
		}
	})
}

func concurrentBench(b *testing.B, concurrency int, iterate func(i int)) {
	semaphore := make(chan bool, concurrency) // concurrency limit

	start := time.Now()
	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		i := i
		semaphore <- true

		go func() {
			defer func() {
				<-semaphore
			}()

			iterate(i)
		}()
	}

	for i := 0; i < cap(semaphore); i++ {
		semaphore <- true
	}

	b.StopTimer()
	elapsed := time.Since(start)

	b.ReportMetric(float64(b.N)/elapsed.Seconds(), "RPS")
}

func failIteration(i int, code int, body []byte) {
	panic(fmt.Sprintf("iteration: %d, unexpected result status: %d, body: %q",
		i, code, string(body)))
}

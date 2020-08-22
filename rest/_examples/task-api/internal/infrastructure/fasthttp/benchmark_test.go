package fasthttp

import (
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/swaggest/fchi"
	"github.com/swaggest/rest/_examples/task-api/internal/domain/task"
	"github.com/swaggest/rest/_examples/task-api/internal/infrastructure"
	"github.com/valyala/fasthttp"
)

// Benchmark_notFound-16    	  987003	      1117 ns/op	    895227 RPS	    1492 B/op	      12 allocs/op
func Benchmark_notFound(b *testing.B) {
	l, err := infrastructure.NewServiceLocator()
	require.NoError(b, err)

	ctx := context.Background()
	tt, err := l.TaskCreator().Create(ctx, task.Value{Goal: "win"})
	require.NoError(b, err)
	assert.Equal(b, 1, tt.ID)

	r := NewRouter(l)

	concurrentBench(b, 50, func(i int) {
		rc := fasthttp.RequestCtx{}
		rc.Request.SetRequestURI("/v0/tasks/1")

		r.ServeHTTP(ctx, &rc)

		if rc.Response.StatusCode() != http.StatusOK {
			failIteration2(i, rc.Response.StatusCode(), rc.Response.Body())
		}
	})
}

//Benchmark_notFound
//Benchmark_notFound-16       	  882198	      1274 ns/op	    785061 RPS	    2175 B/op	      23 allocs/op
//Benchmark_notFoundSrv
//Benchmark_notFoundSrv-16    	   55630	     25398 ns/op	     39353 RPS	   12598 B/op	      94 allocs/op

//Benchmark_notFoundSrv-16    	   76813	     28715 ns/op	     34820 RPS	    8832 B/op	      69 allocs/op
func Benchmark_notFoundSrv(b *testing.B) {
	l, err := infrastructure.NewServiceLocator()
	require.NoError(b, err)

	tt, err := l.TaskCreator().Create(context.Background(), task.Value{Goal: "win"})
	require.NoError(b, err)
	assert.Equal(b, 1, tt.ID)

	r := NewRouter(l)
	srv := fchi.NewTestServer(r)
	defer srv.Close()

	base := srv.URL

	concurrentBench(b, 50, func(i int) {
		getReq, err := http.NewRequest(http.MethodGet, base+"/v0/tasks/1", nil)
		if err != nil {
			b.Fatal(err)
		}

		resp, err := http.DefaultTransport.RoundTrip(getReq)
		if err != nil {
			b.Fatal(err)
		}

		if resp.StatusCode != http.StatusOK {
			failIteration(i, resp)
		}

		_, err = io.Copy(ioutil.Discard, resp.Body)
		if err != nil {
			b.Fatal(err)
		}

		err = resp.Body.Close()
		if err != nil {
			b.Fatal(err)
		}
	})
}

// Benchmark_notFoundSrv2-16    	  447328	      2606 ns/op	    383717 RPS	      16 B/op	       2 allocs/op
func Benchmark_notFoundSrv2(b *testing.B) {
	l, err := infrastructure.NewServiceLocator()
	require.NoError(b, err)

	tt, err := l.TaskCreator().Create(context.Background(), task.Value{Goal: "win"})
	require.NoError(b, err)
	assert.Equal(b, 1, tt.ID)

	r := NewRouter(l)

	srv := fchi.NewTestServer(r)
	defer srv.Close()

	base := srv.URL

	concurrentBench(b, 50, func(i int) {
		req := fasthttp.AcquireRequest()
		resp := fasthttp.AcquireResponse()
		defer fasthttp.ReleaseRequest(req)   // <- do not forget to release
		defer fasthttp.ReleaseResponse(resp) // <- do not forget to release

		req.SetRequestURI(base + "/v0/tasks/1")

		err := fasthttp.Do(req, resp)
		if err != nil {
			b.Fatal(err.Error())
		}

		if resp.StatusCode() != http.StatusOK {
			failIteration2(i, resp.StatusCode(), resp.Body())
		}
	})
}

//Benchmark_invalidBody-16    	  116029	      8762 ns/op	    114108 RPS	    7370 B/op	      74 allocs/op
func Benchmark_invalidBody(b *testing.B) {
	l, err := infrastructure.NewServiceLocator()
	require.NoError(b, err)

	r := NewRouter(l)
	srv := fchi.NewTestServer(r)
	defer srv.Close()

	base := srv.URL

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
		req.SetRequestURI(base + "/v0/tasks/1")
		req.SetBody(body)

		err := fasthttp.Do(req, resp)
		if err != nil {
			b.Fatal(err.Error())
		}

		if resp.StatusCode() != http.StatusBadRequest {
			failIteration2(i, resp.StatusCode(), resp.Body())
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

func failIteration2(i int, code int, body []byte) {
	panic(fmt.Sprintf("iteration: %d, unexpected result status: %d, body: %q",
		i, code, string(body)))
}

func failIteration(i int, res *http.Response) {
	// nolint:errcheck // Ignore unlikely error in benchmark.
	body, _ := ioutil.ReadAll(res.Body)
	panic(fmt.Sprintf("iteration: %d, unexpected result status: %d, body: %q",
		i, res.StatusCode, string(body)))
}

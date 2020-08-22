package main

import (
	"fmt"
	"github.com/swaggest/fchi"
	"net/http"
	"testing"
	"time"

	"github.com/valyala/fasthttp"
)

func Benchmark_newRouter_something(b *testing.B) {
	r := newRouter()
	go fasthttp.ListenAndServe(":8085", fchi.RequestHandler(r))

	concurrentBench(b, 50, func(i int) {
		req := fasthttp.AcquireRequest()
		resp := fasthttp.AcquireResponse()
		defer fasthttp.ReleaseRequest(req)   // <- do not forget to release
		defer fasthttp.ReleaseResponse(resp) // <- do not forget to release

		req.SetRequestURI("http://localhost:8085/something")

		err := fasthttp.Do(req, resp)
		if err != nil {
			b.Fatal(err.Error())
		}

		if resp.StatusCode() != http.StatusOK {
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

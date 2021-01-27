package main

import (
	"encoding/json"
	"fmt"
	"github.com/gohp/goutils/convert"
	"github.com/spf13/pflag"
	"golang.org/x/net/websocket"
	"log"
	"sync"
	"sync/atomic"
	"time"
)

const _verifyResponse = "pong"

var (
	host, url string
	port int
	threadNums int
	msgNums int
	count int64
	successCount int64
	wg sync.WaitGroup
	results []Result
	resultCh chan Result
)

type Result struct {
	latency int64
}

type wsResponse struct {
	Msg string `json:"msg"`
}

func init()  {
	pflag.StringVarP(&host, "host", "h", "127.0.0.1", "default")
	pflag.IntVarP(&port, "port", "p", 2000, "default")
	pflag.IntVarP(&threadNums, "threadNums","t", 10, "default")
	pflag.IntVarP(&msgNums, "msgNums", "m", 1, "default")
}

func worker(threadId int64)  {
	ws, err := websocket.Dial(url, "", "https://baidu.com")
	if err != nil {
		log.Println("worker dial fail:", err)
		wg.Done()
		return
	}

	defer func() {
		ws.Close()
		wg.Done()
	}()

	for i := 0; i < msgNums; i ++ {
		startTime := time.Now()
		msg := []byte("{\"msg\":\"ping\", \"id\":\"" + convert.Int64ToString(threadId) + "\"}")
		atomic.AddInt64(&count, 1)
		_, err := ws.Write(msg)
		if err != nil {
			return
		}
		isVerify := verifyResponse(ws)
		if isVerify {
			resultCh <- Result{
				latency: int64(time.Since(startTime)),
			}
			atomic.AddInt64(&successCount, 1)
		}
	}
}

func verifyResponse(ws *websocket.Conn) (bool) {
	msg := make([]byte, 256)
	n, err := ws.Read(msg)

	var r wsResponse
	err = json.Unmarshal(msg[:n], &r)
	if err != nil {
		return false
	}

	if r.Msg != _verifyResponse {
		return false
	}
	return true
}

func saveResult() {
	for {
		select {
		case r := <- resultCh:
			results = append(results, r)
		}
	}
}

func stat() (maxLatF, minLatF, avgLatF float64) {
	var (
		c int64 = 0
		maxLat, minLat, avgLat int64

	)
	if len(results) == 0 {
		return
	}
	for _, e := range results {
		t := e.latency

		if t >= maxLat {
			maxLat = t
		}

		if minLat == 0  || t <= minLat {
			minLat = t
		}
		avgLat += t
		c += 1
	}

	avgLat /= c

	// 纳秒=>毫秒
	maxLatF = float64(maxLat) / 1e6
	minLatF = float64(minLat) / 1e6
	avgLatF = float64(avgLat) / 1e6
	return
}

func main()  {
	pflag.Parse()
	resultCh = make(chan Result, 1000)
	results = make([]Result, 0)
	log.Printf("test ws://%s:%d %d * %d\n", host, port, threadNums, msgNums)
	url = fmt.Sprintf("ws://%s:%d", host, port)

	go saveResult()

	wg.Add(threadNums)
	for i := 0; i < threadNums; i ++ {
		go worker(int64(i))
	}

	wg.Wait()

	maxLat, minLat, avgLat  := stat()

	log.Println("done")
	log.Printf("count: %d, success: %d\n", count, successCount)
	log.Printf("maxLat: %8.2fms, minLat: %8.2fms, avgLat: %8.2fms\n", maxLat, minLat, avgLat)
}


 
package main

import (
	"fmt"
	"sync"
	"net/http"
	"os"
	"strings"
	"bufio"
	"time"
	//"encoding/json"
)

type Req struct {
	Host	string
	File    string
	Range	string
	Method	string
	Delay	int64
}

type Results struct {
	RetCode map[string]int
	XCache map[string]int
}

const (
	CLFTIME = "02/Jan/2006:15:04:05"
)

func worker(requests <-chan *Req, wg *sync.WaitGroup, results *Results) {
	defer wg.Done()
	for request := range requests {
		target := "http://edge20-vir-n.maxcdn.net"
		url := fmt.Sprintf("%s%s", target, request.File)

		req, err :=  http.NewRequest(request.Method, url, nil)
		req.Host = request.Host
		if request.Range != "-" {
			req.Header.Set("Range:", request.Range)
		}

		time.Sleep(time.Duration(request.Delay) * time.Second)
		client := &http.Client{}
		resp, err := client.Do(req)
		if err != nil {panic(err)}

		results.RetCode[resp.Status] += 1
		if len(resp.Header["X-Cache"]) > 0 {
			results.XCache[resp.Header["X-Cache"][0]] += 1
		}
		resp.Body.Close()
	}
}

func main() {

	var wg sync.WaitGroup
	requests := make(chan *Req)

	var totalrequests int
	results := &Results{}
	results.RetCode = make(map[string]int)
	results.XCache = make(map[string]int)

	for i := 0; i < 10; i++ {
		wg.Add(1)
		go worker(requests, &wg, results)
	}

	f, err := os.Open("nginx.log")
	if err != nil {panic(err)}
	scanner := bufio.NewScanner(f)
	var basetime, delay int64

	for scanner.Scan() {
		line := strings.Split(scanner.Text(), " ")

		t, _ := time.Parse(CLFTIME, strings.Trim(line[2], "["))
		if totalrequests == 0 {
			basetime = t.Unix()
		}
		delay = t.Unix() - basetime

		requests <- &Req{
			line[1],
			line[5],
			line[11],
			strings.Trim(line[4], "\""),
			delay,
		}
		totalrequests += 1
	}
	f.Close()

	close(requests)
	wg.Wait()

	fmt.Printf("total requests completed: %d\n", totalrequests)
	for rcode, count := range results.RetCode {
		fmt.Printf("return code: %s with ratio %.2f\n", rcode, float32(count)/float32(totalrequests))
	}
	for hitormiss, count := range results.XCache {
		fmt.Printf("X-Cache %s: ratio %.2f\n", hitormiss, float32(count)/float32(totalrequests))
	}
}
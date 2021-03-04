package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"
)

var (
	urlRE      = regexp.MustCompile(`https?:\/\/(www\.)?[-a-zA-Z0-9@:%._\+~#=]{1,256}\.[a-zA-Z0-9()]{1,6}\b([-a-zA-Z0-9()!@:%_\+.~#?&\/\/=]*)`)
	skipStatus = flag.String("a", "", "-a 500,400")
	timeout    = flag.Duration("t", 5*time.Second, "-t 10")
	whitelist  = flag.String("w", "", "-w server1.com,server2.com")
)

const (
	okColor      = "\033[1;34m%s\033[0m\n"
	warningColor = "\033[1;33m%s\033[0m\n"
	errorColor   = "\033[1;31m%s\033[0m\n"
	debugColor   = "\033[0;36m%s\033[0m\n"
)

type response struct {
	URL      string
	Response *http.Response
	Err      error
}

func main() {
	flag.Parse()

	args := flag.Args()
	if len(args) == 0 {
		log.Fatal("filename is required")
	}

	// read file
	file, err := ioutil.ReadFile(args[0])
	if err != nil {
		log.Fatalf("error on reading file: %v", err)
	}

	// validate skipStatus
	var skipped []int
	if len(*skipStatus) > 0 {
		splitted := strings.Split(*skipStatus, ",")
		for _, item := range splitted {
			val, err := strconv.Atoi(item)
			if err != nil {
				log.Fatalf("could not parse skip status value: %v \n", err)
			}
			skipped = append(skipped, val)
		}
	}

	// validate whitelist
	var whitelisted []string
	if len(*whitelist) > 0 {
		whitelisted = strings.Split(*whitelist, ",")
	}

	matches := urlRE.FindAllString(string(file), -1)
	client := &http.Client{
		Timeout: *timeout,
	}
	wg := &sync.WaitGroup{}

	results := make(chan *response)
	requests := make(chan string)

	// spawn workers
	for i := 0; i <= 50; i++ {
		wg.Add(1)
		go worker(wg, requests, results, client)
	}

	go func() {
		for _, url := range matches {
			if isInStr(url, whitelisted) {
				continue
			}
			requests <- url
		}
		close(requests)
		wg.Wait()
		close(results)
	}()

	for i := range results {
		shouldSkipURL := len(skipped) > 0 && isIn(i.Response.StatusCode, skipped)
		if i.Err != nil {
			fmt.Printf(errorColor, fmt.Sprintf("[ERROR] %s", i.URL))
			continue
		}

		statusColor := okColor
		if i.Response.StatusCode > 400 {
			statusColor = errorColor
		}

		if !shouldSkipURL {
			fmt.Printf(statusColor, fmt.Sprintf("[%d] %s", i.Response.StatusCode, i.URL))
		} else {
			fmt.Printf(debugColor, fmt.Sprintf("[%d] %s", i.Response.StatusCode, i.URL))
		}
	}
}

func worker(wg *sync.WaitGroup, requests <-chan string, results chan<- *response, client *http.Client) {
	wg.Done()
	for url := range requests {
		response := &response{
			URL: url,
		}
		req, err := http.NewRequest("HEAD", url, nil)
		if err != nil {
			response.Err = err
			results <- response
			continue
		}

		resp, err := client.Do(req)
		if err != nil {
			response.Err = err
			results <- response
			continue
		}

		response.Response = resp
		results <- response
	}
}

func isIn(item int, items []int) bool {
	for _, i := range items {
		if i == item {
			return true
		}
	}
	return false
}

func isInStr(item string, items []string) bool {
	for _, i := range items {
		if i == item {
			return true
		}
	}
	return false
}

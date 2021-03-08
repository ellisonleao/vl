package main

import (
	"crypto/tls"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"
)

var (
	urlRE      = regexp.MustCompile(`https?:\/\/(www\.)?[-a-zA-Z0-9@:%._\+~#=]{1,256}\.[a-zA-Z0-9]{1,6}\b([-a-zA-Z0-9!@:%_\+.~#?&\/\/=]*)`)
	skipStatus = flag.String("a", "", "-a 500,400")
	timeout    = flag.Duration("t", 5*time.Second, "-t 10s or -t 1h")
	whitelist  = flag.String("w", "", "-w server1.com,server2.com")
	size       = flag.Int("s", 50, "-s 50")
)

var (
	errorColor    = "\033[1;31m%d\033[0m"
	errorStrColor = "\033[1;31m%s\033[0m"
	okColor       = "\033[1;32m%d\033[0m"
	debugColor    = "\033[1;36m%d\033[0m"
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
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		},
	}
	var wg sync.WaitGroup

	results := make(chan *response)
	requests := make(chan string)

	// spawn workers
	for i := 0; i <= *size; i++ {
		wg.Add(1)
		go worker(&wg, requests, results, client)
	}

	// producer
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
		fmt.Printf("Found %d URIs\n", len(matches))
	}()

	totalErrors := 0
	for i := range results {
		if i.Err != nil {
			fmt.Printf("[%s] %s\n", fmt.Sprintf(errorStrColor, "ERROR"), i.URL)
			totalErrors++
			continue
		}

		shouldSkipURL := len(skipped) > 0 && isIn(i.Response.StatusCode, skipped)
		statusColor := okColor
		if i.Response.StatusCode > 400 && !shouldSkipURL {
			statusColor = errorColor
			totalErrors++
		} else if shouldSkipURL {
			statusColor = debugColor
		}

		fmt.Printf("[%s] %s \n", fmt.Sprintf(statusColor, i.Response.StatusCode), i.URL)
	}
	if totalErrors > 0 {
		fmt.Printf("Total Errors: %s \n", fmt.Sprintf(errorColor, totalErrors))
		os.Exit(1)
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

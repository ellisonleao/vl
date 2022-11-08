package main

import (
	"crypto/tls"
	"flag"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"
)

var (
	urlRE           = regexp.MustCompile(`https?:\/\/(www\.)?[-a-zA-Z0-9@:%._\+~#=]{1,256}\.[a-zA-Z0-9]{1,6}\b([-a-zA-Z0-9!@:%_\+.~#?&\/\/=$]*)`)
	skipStatus      = flag.String("a", "", "-a 500,400")
	timeout         = flag.Duration("t", 10*time.Second, "-t 10s or -t 1h")
	whitelist       = flag.String("w", "", "-w server1.com,server2.com")
	s               = rand.NewSource(time.Now().Unix())
	backoffSchedule = []time.Duration{
		1 * time.Second,
		3 * time.Second,
		5 * time.Second,
	}
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
	file, err := os.ReadFile(args[0])
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

	results := make(chan *response)

	// producer
	counter := 0
	for _, url := range matches {
		u := url
		if isIn(u, whitelisted) {
			continue
		}
		counter++
		go worker(u, results, client)
	}
	fmt.Printf("Found %d URIs\n", len(matches))

	totalErrors := 0
	for counter > 0 {
		resp := <-results
		counter--
		if resp.Err != nil && resp.Response == nil {
			fmt.Printf("[%s] %s\n", fmt.Sprintf(errorStrColor, "ERROR"), resp.Err.Error())
			totalErrors++
			continue
		}

		shouldSkipURL := len(skipped) > 0 && isIn(resp.Response.StatusCode, skipped)
		statusColor := okColor
		if resp.Response.StatusCode > http.StatusBadRequest && !shouldSkipURL {
			statusColor = errorColor
			totalErrors++
		} else if shouldSkipURL {
			statusColor = debugColor
		}

		fmt.Printf("[%s] %s \n", fmt.Sprintf(statusColor, resp.Response.StatusCode), resp.URL)
	}

	if totalErrors > 0 {
		fmt.Printf("Total Errors: %s \n", fmt.Sprintf(errorColor, totalErrors))
		os.Exit(1)
	}
}

func newRequest(url string) (*http.Request, error) {
	userAgents := []string{
		"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_5) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/13.1.1 Safari/605.1.15",
		"Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:77.0) Gecko/20100101 Firefox/77.0",
		"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_5) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/83.0.4103.97 Safari/537.36",
		"Mozilla/5.0 (Macintosh; Intel Mac OS X 10.15; rv:77.0) Gecko/20100101 Firefox/77.0",
		"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/83.0.4103.97 Safari/537.36",
	}

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	userAgent := userAgents[rand.Intn(len(userAgents))]

	req.Header.Add("User-Agent", userAgent)

	return req, err
}

func worker(url string, results chan<- *response, client *http.Client) {
	var err error

	response := &response{
		URL: url,
	}

	req, err := newRequest(url)
	if err != nil {
		response.Err = err
		return
	}

	resp, err := client.Do(req)
	if err != nil {
		for _, backoff := range backoffSchedule {
			time.Sleep(backoff)

			// trying a new request with a different user-agent
			req, err := newRequest(url)
			if err != nil {
				response.Err = err
				break
			}

			resp, err := client.Do(req)
			if err == nil {
				response.Response = resp
				break
			}
		}
	}

	response.Response = resp
	response.Err = err
	results <- response
}

func isIn[item int | string](val item, values []item) bool {
	for _, i := range values {
		if i == val {
			return true
		}
	}
	return false
}

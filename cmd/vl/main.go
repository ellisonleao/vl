package main

import (
	"crypto/tls"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"
)

var (
	urlRE      = regexp.MustCompile(`https?:\/\/(www\.)?[-a-zA-Z0-9@:%._\+~#=]{1,256}\.[a-zA-Z0-9]{1,6}\b([-a-zA-Z0-9!@:%_\+.~#?&\/\/=$]*)`)
	skipStatus = flag.String("a", "", "-a 500,400")
	timeout    = flag.Duration("t", 10*time.Second, "-t 10s or -t 1h")
	whitelist  = flag.String("w", "", "-w server1.com,server2.com")
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
		if isInStr(url, whitelisted) {
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
		if resp.Response.StatusCode > 400 && !shouldSkipURL {
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

func worker(url string, results chan<- *response, client *http.Client) {
	response := &response{
		URL: url,
	}
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		response.Err = err
		return
	}

	resp, err := client.Do(req)
	if err != nil {
		response.Err = err
		results <- response
		return
	}
	defer resp.Body.Close()

	response.Response = resp
	results <- response
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

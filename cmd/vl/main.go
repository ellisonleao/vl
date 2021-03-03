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
	OKColor      = "\033[1;34m%s\033[0m\n"
	WarningColor = "\033[1;33m%s\033[0m\n"
	ErrorColor   = "\033[1;31m%s\033[0m\n"
	DebugColor   = "\033[0;36m%s\033[0m\n"
)

func worker(wg *sync.WaitGroup, url string, skipped []int, client *http.Client) {
	defer wg.Done()

	req, err := http.NewRequest("OPTIONS", url, nil)
	if err != nil {
		log.Fatalf("error on creating request: %v", err)
	}

	resp, err := client.Do(req)
	if err != nil {
		fmt.Printf(ErrorColor, "[ERROR] "+url)
		return
	}

	shouldSkipURL := len(skipped) > 0 && isIn(resp.StatusCode, skipped)
	isError := resp.StatusCode > 400
	if !shouldSkipURL {
		statusColor := OKColor
		if isError {
			statusColor = ErrorColor
		}
		fmt.Printf(statusColor, fmt.Sprintf("[%d] %s", resp.StatusCode, url))
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

	for _, url := range matches {
		wg.Add(1)
		if isInStr(url, whitelisted) {
			continue
		}
		go worker(wg, url, skipped, client)
	}
	wg.Wait()
}

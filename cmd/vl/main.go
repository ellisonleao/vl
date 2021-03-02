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
	"time"
)

var (
	urlRE      = regexp.MustCompile(`https?:\/\/(www\.)?[-a-zA-Z0-9@:%._\+~#=]{1,256}\.[a-zA-Z0-9()]{1,6}\b([-a-zA-Z0-9()!@:%_\+.~#?&\/\/=]*)`)
	skipStatus = flag.String("a", "", "-a 500,400")
	timeout    = flag.Duration("t", 3*time.Second, "-t 10")
)

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

	isIn := func(val int, items []int) bool {
		for _, i := range items {
			if i == val {
				return true
			}
		}
		return false
	}

	matches := urlRE.FindAllString(string(file), -1)
	client := &http.Client{
		Timeout: *timeout,
	}

	for _, url := range matches {
		resp, err := client.Get(url)
		if err != nil {
			fmt.Printf("Error on getting url %s: %v \n", url, err)
			continue
		}

		if len(skipped) > 0 && isIn(resp.StatusCode, skipped) {
			continue
		}

		fmt.Printf("url=%s status=%d \n", url, resp.StatusCode)
	}

}

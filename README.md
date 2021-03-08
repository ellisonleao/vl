# verify-links

CLI tool that helps verify current status of URIs in text files

# Installing

```
go get github.com/npxbr/verify-links/cmd/vl
```

# Usage

```
  -a string (Skip status code list)
        -a 500,400
  -s int (Concurrency size)
        -s 50 (default 50)
  -t duration (max timeout for each request)
        -t 10s or -t 1h (default 5s)
  -w string (whitelist URI list)
        -w server1.com,server2.com
```

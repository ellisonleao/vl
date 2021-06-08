FROM golang:1.16.5-stretch AS build
WORKDIR /
COPY . . 
RUN CGO_ENABLED=0 go build -ldflags='-s -w' -o /vl ./cmd/vl/main.go 

FROM scratch
WORKDIR /bin
COPY --from=build /vl /bin/vl
ENTRYPOINT ["./vl"]

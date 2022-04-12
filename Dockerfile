# build
FROM golang:1.16-alpine as builder

RUN apk add --no-cache bash

WORKDIR /go/src/cubone

COPY go.mod .
COPY go.sum .
COPY cmd/ ./cmd
COPY *.go .

# set the -s and -w linker flags to strip the debugging information
RUN env GO111MODULE=on CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o cubone cmd/main.go

# deploy
FROM alpine:3.12 as production

WORKDIR /app

COPY --from=builder /go/src/cubone/cubone ./

RUN chmod +x /app/cubone

ENTRYPOINT /app/cubone


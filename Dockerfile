FROM golang:1.15 AS builder

WORKDIR /src
COPY ./ ./
RUN go build -o app

ENTRYPOINT ["./app"]

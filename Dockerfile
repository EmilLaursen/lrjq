FROM golang:1.17-alpine as builder

ENV GO111MODULE=on

WORKDIR /app

COPY go.mod go.sum ./

RUN go install github.com/cespare/reflex@latest
RUN go mod download

COPY openapi/ openapi/
COPY cmd/ cmd/
COPY src/ src/

ENV CGO_ENABLED 0
RUN GOOS=linux GOARCH=amd64 go build \
      -ldflags='-w -s -extldflags "-static"' -a \
      -o /app/queue_server cmd/main.go

FROM gcr.io/distroless/static as production
COPY --from=builder /app/queue_server /
ENTRYPOINT ["/queue_server"]

FROM golang:1.17-alpine as development

ENV CGO_ENABLED 0
COPY --from=builder /app/queue_server /
WORKDIR /app
COPY --from=builder /go/bin/reflex /usr/bin/reflex
ENTRYPOINT ["reflex", "-sr", ".*(\\.yaml|\\.go)$$", "--", "go", "run", "cmd/main.go"]

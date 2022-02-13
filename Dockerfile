FROM golang:1.17-bullseye as builder

ENV GO111MODULE=on

WORKDIR /app

COPY go.mod go.sum ./

RUN go mod download

COPY openapi/ openapi/
COPY cmd/ cmd/
COPY src/ src/

RUN go build -o queue_server cmd/main.go

FROM gcr.io/distroless/base as production

COPY --from=builder /app/queue_server /

CMD ["/queue_server"]

FROM golang:1.17-bullseye as development

WORKDIR /app
RUN go install github.com/cespare/reflex@latest
ENTRYPOINT ["/go/bin/reflex", "-vsr", ".*\.yaml|.*\.go", "--", "sh", "-c", "go run /cmd/main.go"]

# CMD ["/queue_server"]

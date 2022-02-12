FROM golang:1.17 as builder

ENV GO111MODULE=on

WORKDIR /app

COPY go.mod go.sum ./

RUN go mod download

COPY openapi/ openapi/
COPY cmd/ cmd/
COPY src/ src/

RUN go build -o queue_server cmd/main.go

FROM gcr.io/distroless/base

COPY --from=builder /app/queue_server /

CMD ["/queue_server"]

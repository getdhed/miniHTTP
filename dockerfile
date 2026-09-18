FROM golang:1.26.2-bookworm AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o server .

FROM debian:bookworm-slim AS api

WORKDIR /app
COPY --from=builder /app/server ./server
EXPOSE 3030
CMD ["./server"]

FROM golang:1.26.2-bookworm AS migrator

RUN go install github.com/pressly/goose/v3/cmd/goose@v3.27.0
WORKDIR /app
COPY migrations ./migrations
ENTRYPOINT ["/go/bin/goose"]
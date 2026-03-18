FROM golang:alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /bot ./cmd/bot
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /email-watcher ./cmd/email-watcher

FROM alpine:3.19 AS bot
RUN apk add --no-cache ca-certificates tzdata
COPY --from=builder /bot /bot
COPY migrations/ /migrations/
ENTRYPOINT ["/bot"]

FROM alpine:3.19 AS email-watcher
RUN apk add --no-cache ca-certificates tzdata
COPY --from=builder /email-watcher /email-watcher
ENTRYPOINT ["/email-watcher"]

FROM golang:1.24-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
ENV CGO_ENABLED=0 GOOS=linux GOARCH=amd64
RUN go build -ldflags="-s -w" -o app ./cmd/server

# ---------- Runtime Stage ----------
FROM alpine:3.19 AS final

WORKDIR /app

RUN apk add --no-cache su-exec \
    && adduser -D appuser \
    && mkdir -p /data && chown appuser:appuser /data

# Copy binary and entrypoint
COPY --from=builder /app/app .
COPY entrypoint.sh .
RUN chmod +x entrypoint.sh

ENTRYPOINT ["./entrypoint.sh"]
CMD ["./app"]

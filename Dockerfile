FROM golang:1.22-alpine AS builder

WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -ldflags="-s -w" -o /firewatch ./cmd/server

FROM alpine:3.19
RUN apk add --no-cache ca-certificates
COPY --from=builder /firewatch /firewatch
COPY static/ /static/

ENV STATIC_DIR=/static
EXPOSE 8080

ENTRYPOINT ["/firewatch"]

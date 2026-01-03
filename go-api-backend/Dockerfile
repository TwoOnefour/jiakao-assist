FROM golang:1.21-alpine AS builder
WORKDIR /app
COPY .. .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o jiakao cmd/backend/main.go
FROM alpine:latest
WORKDIR /app
COPY --from=builder /app/jiakao .
EXPOSE 8080
CMD ["./jiakao"]

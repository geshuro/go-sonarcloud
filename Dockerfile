# Build stage
FROM golang:1.23.5-alpine AS builder
WORKDIR /app
COPY go.mod ./
COPY go.sum ./
RUN go mod download
COPY . .
RUN GOOS=linux GOARCH=amd64 go build -o /tmp/app

# Final image
FROM alpine:latest
COPY --from=builder /tmp/app /app
CMD [ "/app" ]
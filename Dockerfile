# Build stage
FROM golang:alpine AS builder
WORKDIR /app

# Install git and other dependencies if needed (alpine)
RUN apk add --no-cache git

COPY go.mod go.sum ./
ENV GOPROXY=https://goproxy.cn,direct
RUN go mod download

COPY . .
RUN go build -o server ./main.go

# Run stage
FROM alpine:latest
WORKDIR /app

# Install tzdata for timezones
RUN apk add --no-cache tzdata

COPY --from=builder /app/server .
COPY configs ./configs
COPY web ./web

# Create download directory
RUN mkdir -p download

EXPOSE 8080

ENV ENV=prod

CMD ["./server"]

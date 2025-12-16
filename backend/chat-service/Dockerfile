# 1. Aşama: Build
FROM golang:1.25-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN go build -o main .

# 2. Aşama: Run
FROM alpine:latest
WORKDIR /root/
COPY --from=builder /app/main .

EXPOSE 8081 

CMD ["./main"]
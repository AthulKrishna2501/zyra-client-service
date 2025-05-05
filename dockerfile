# syntax=docker/dockerfile:1
FROM golang:1.24 AS builder

WORKDIR /app
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o main ./cmd

# Final image
FROM alpine:3.18  
WORKDIR /root/
COPY --from=builder /app/main .
EXPOSE 5005
CMD ["./main"]

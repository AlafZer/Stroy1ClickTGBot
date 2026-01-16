FROM golang:latest AS builder

WORKDIR /src

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /app/notifTelegramApp ./main

FROM alpine:latest
WORKDIR /app

COPY --from=builder /app/notifTelegramApp /app/
CMD ["/app/notifTelegramApp"]
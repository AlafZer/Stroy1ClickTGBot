FROM golang:latest AS builder

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /app/notifTelegramApp ./main
RUN ls ../src

FROM alpine:latest
WORKDIR /app

COPY --from=builder /app/notifTelegramApp /app/
COPY --from=builder /src/docs/ /app/docs

CMD ["/app/notifTelegramApp"]
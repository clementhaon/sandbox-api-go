FROM golang:1.26.1-alpine3.23 AS builder

RUN apk add --no-cache git ca-certificates

WORKDIR /app

COPY pkg/ ./pkg/
COPY services/notification-service/ ./services/notification-service/

WORKDIR /app/services/notification-service
RUN go mod download && CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o /app/main .

FROM alpine:latest
RUN apk upgrade --no-cache && apk add --no-cache ca-certificates
RUN addgroup -g 1001 -S appgroup && adduser -u 1001 -S appuser -G appgroup

WORKDIR /root/
COPY --from=builder /app/main .
COPY --from=builder /app/services/notification-service/database/migrations ./database/migrations

RUN chown -R appuser:appgroup /root/
USER appuser

EXPOSE 8083
CMD ["./main"]

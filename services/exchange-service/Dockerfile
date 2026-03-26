FROM golang:1.26-alpine AS builder
WORKDIR /app
COPY . .
RUN go build -o /service ./services/exchange-service

FROM alpine:3.20
COPY --from=builder /service /service
ENTRYPOINT ["/service"]

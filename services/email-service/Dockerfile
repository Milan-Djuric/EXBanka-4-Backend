FROM golang:1.26-alpine AS builder
WORKDIR /app
COPY . .
RUN go build -o /service ./services/email-service

FROM alpine:3.20
WORKDIR /app
COPY --from=builder /service /app/service
COPY services/email-service/templates/ /app/templates/
ENTRYPOINT ["/app/service"]

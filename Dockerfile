FROM golang:1.21.5-alpine AS builder
WORKDIR /app
COPY . .
RUN go build -o main ./cmd/main.go

FROM alpine:latest
RUN apk --no-cache add ca-certificates aws-cli

WORKDIR /app
COPY --from=builder /app/main .

RUN mkdir -p /app/pkg/config/envs

COPY startup.sh /startup.sh
RUN chmod +x /startup.sh

ENTRYPOINT ["/startup.sh"]
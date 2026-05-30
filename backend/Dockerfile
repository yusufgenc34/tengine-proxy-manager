FROM golang:1.25-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go build -o tpm ./cmd/server/

FROM alpine:3.19
RUN apk --no-cache add ca-certificates tzdata certbot
WORKDIR /root/
COPY --from=builder /app/tpm .
COPY --from=builder /app/templates ./templates
EXPOSE 4000
CMD ["./tpm"]

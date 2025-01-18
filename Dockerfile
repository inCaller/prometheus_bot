FROM golang:1.23.5-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN GOGC=off CGO_ENABLED=0 go build -v -o prometheus_bot


FROM alpine:3.21 as alpine
RUN apk add --no-cache ca-certificates tzdata


FROM scratch
EXPOSE 9087
WORKDIR /
COPY --from=alpine /etc/passwd /etc/group /etc/
COPY --from=alpine /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=alpine /usr/share/zoneinfo /usr/share/zoneinfo
COPY --from=builder /app/prometheus_bot /prometheus_bot
USER nobody
CMD ["/prometheus_bot"]

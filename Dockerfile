FROM golang:1.17.6-alpine3.15 as builder
RUN addgroup -S appgroup && adduser -S appuser -G appgroup -u 1000
RUN apk add --update --no-cache ca-certificates tzdata
WORKDIR /app
COPY go.mod .
COPY go.sum .
RUN go mod download
COPY . .
RUN GOGC=off CGO_ENABLED=0 go build -v -o prometheus_bot

FROM scratch
COPY --from=builder /usr/share/zoneinfo /usr/share/zoneinfo
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder \
  /etc/passwd \
  /etc/group \
 /etc/
WORKDIR /
COPY --from=builder /app/prometheus_bot .
USER appuser
EXPOSE 9087
CMD ["/prometheus_bot"]

FROM golang:1.17.3-alpine3.14 as builder
RUN apk add --no-cache git ca-certificates make tzdata
COPY . /app
RUN cd /app && \
    go get -d -v && \
    CGO_ENABLED=0 GOOS=linux go build -v -a -installsuffix cgo -o prometheus_bot

FROM alpine:3.14
COPY --from=builder /app/prometheus_bot /
RUN apk add --no-cache ca-certificates tzdata tini
USER nobody
EXPOSE 9087
ENTRYPOINT ["/sbin/tini", "--"]
CMD ["/prometheus_bot"]

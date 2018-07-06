FROM golang:1.10.2 as builder
WORKDIR /
RUN git clone https://github.com/inCaller/prometheus_bot && \
    cd prometheus_bot && \
    go get -d -v && \
    CGO_ENABLED=0 GOOS=linux go build -v -a -installsuffix cgo -o prometheus_bot


FROM alpine:3.5
COPY --from=builder /prometheus_bot/prometheus_bot /
RUN apk add --no-cache ca-certificates
EXPOSE 9087
ENTRYPOINT ["/prometheus_bot"]

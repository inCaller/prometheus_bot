FROM alpine:3.5

RUN apk add --no-cache ca-certificates

COPY prometheus_bot /

EXPOSE 9087
ENTRYPOINT ["/prometheus_bot"]

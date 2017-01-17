FROM golang

add prometheus_bot /prometheus_bot/prometheus_bot
COPY config.yaml /prometheus_bot/config.yaml

EXPOSE      9087

WORKDIR /prometheus_bot

CMD ["./prometheus_bot"]

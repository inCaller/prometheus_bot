FROM scratch
EXPOSE 9087
COPY prometheus_bot /
ENTRYPOINT ["/prometheus_bot"]

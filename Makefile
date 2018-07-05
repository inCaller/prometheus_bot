TARGET=prometheus_bot

all: main.go
	go build -o $(TARGET)
test: all
	prove -v
clean:
	go clean
	rm -f $(TARGET)
	rm -f bot.log

install_dependencies:
	go get github.com/gin-gonic/gin
	go get github.com/go-telegram-bot-api/telegram-bot-api

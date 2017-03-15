TARGET=prometheus_bot

all: main.go
	go build -o $(TARGET)
test:
	./prometheus_bot > bot.log 2>&1 &
	sleep 3
	prove -v
	jobs -p | xargs kill
clean:
	rm $(TARGET)

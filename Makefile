TARGET=prometheus_bot

all: main.go
	go build -o $(TARGET)
test:
	prove -v
clean:
	go clean
	rm -f $(TARGET)

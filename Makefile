TARGET=prometheus_bot

all: main.go
	go build -o $(TARGET)
test:
	prove -v
clean:
	rm $(TARGET)

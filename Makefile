TARGET=prometheus_bot


all: main.go
	go build -o $(TARGET)
clean:
	rm $(TARGET)

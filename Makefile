TARGET=prometheus_bot


all: main.go
	go build -o $(TARGET)
test:
	bash t/curl.t
clean:
	rm $(TARGET)

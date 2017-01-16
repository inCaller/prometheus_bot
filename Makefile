TARGET=$(shell pwd)/prometheus_bot


all: build

build:
	@echo ">> building go file"
	@go build -o $(TARGET)
docker:
	@echo ">> building docker image"
	@docker build -t prometheus_bot .
test:
	bash t/curl.t
clean:
	rm $(TARGET)

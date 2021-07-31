build:
	go build -o bin/chattin-server src/*.go

run:
	go run -race src/*.go

docker:
	docker build . -t chattin-server

clean:
	rm -rf bin

all: build
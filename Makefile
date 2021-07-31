build:
	go build -o bin/chattin-server src/*.go

run:
	go run -race src/*.go

clean:
	rm -rf server

all: build
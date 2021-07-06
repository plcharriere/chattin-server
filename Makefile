build:
	go build -o server.out src/*.go

run:
	go run -race src/*.go

clean:
	rm -rf server

all: build
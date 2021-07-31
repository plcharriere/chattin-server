# Chattin Server

## Run

`make run`

Run the server without building a binary.

## Build

`make build`

Build a binary in `bin/`

## Run in a Docker container

First, you need to build the Docker image with `make docker`

Then you can run the container with :

```
docker run --restart always -d --name chattin-server \
  -e ADDRESS=0.0.0.0:2727 \
  -v /path/to/cert:/chattin-server/ssl/cert \
  -v /path/to/key:/chattin-server/ssl/key \
  -e SSL_CERT=/chattin-server/ssl/cert \
  -e SSL_KEY=/chattin-server/ssl/key \
  -e POSTGRES_ADDRESS=127.0.0.1:5432 \
  -e POSTGRES_USER=postgres \
  -e POSTGRES_PASSWORD=password \
  -e POSTGRES_DATABASE=chattin \
  chattin-server
```

SSL related lines are optionals.

## Run in a Docker container with Docker-Compose

First, you need to build the Docker image with `make docker`

Then write in a `docker-compose.yml` file :

```
version: "3.6"

services:
  chattin-server:
    image: chattin-server
    volumes:
      - /path/to/cert:/chattin-server/ssl/cert
      - /path/to/key:/chattin-server/ssl/key
    environment:
      - ADDRESS=:2727
      - SSL_CERT=/chattin-server/ssl/cert
      - SSL_KEY=/chattin-server/ssl/key
      - POSTGRES_ADDRESS=postgres:5432
      - POSTGRES_USER=postgres
      - POSTGRES_PASSWORD=ch4ng3_m3
      - POSTGRES_DATABASE=chattin
    ports:
      - 2727:2727

  postgres:
    image: postgres
    volumes:
      - ./postgres/data:/var/lib/postgresql/data
    environment:
      - POSTGRES_USER=postgres
      - POSTGRES_PASSWORD=ch4ng3_m3
      - POSTGRES_DB=chattin
```

Then after configuring your `docker-compose.yml` file, you can run :

`docker-compose up -d`
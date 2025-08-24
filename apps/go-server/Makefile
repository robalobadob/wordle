run:
	go run ./...

build:
	go build -o bin/go-server .

docker-build:
	docker build -t wordle/go-server:dev .

docker-run:
	docker run --rm -p 5175:5175 --env-file .env wordle/go-server:dev

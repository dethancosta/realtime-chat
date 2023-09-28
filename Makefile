server:
	brew services start redis
	go run ./cmd/server

client:
	go run ./cmd/client

compose-up:
	docker-compose up --build -d

compose-down:
	docker-compose down

protoc-generate:
	protoc \
		--go_out=./proto/generated --go-grpc_out=./proto/generated \
		--go-grpc_opt=paths=source_relative --go_opt=paths=source_relative \
		--proto_path=./proto ./proto/events.proto
	go mod download
build:
	./build.sh

run: build
	go run main.go serve --config app.config.yaml

serve:
	docker-compose down
	docker-compose up -d

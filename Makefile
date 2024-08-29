BINARY_NAME=hitbox-phole
DOCKER_IMAGE_NAME=hitbox-platform-api
DOCKER_TAG=latest

swagger:
	swag init -g cmd/main.go --parseDependency --parseInternal

github-swagger:
	cd cmd && swag init -g main.go --parseDependency --parseInternal -d ./

generate-sdk:
	cd docs && openapi-generator-cli generate -i swagger3.json -g typescript-axios -o ./client-sdk

convert-swagger:
	@echo "Converting Swagger to OpenAPI 3.0..."
	@curl -X POST "https://converter.swagger.io/api/convert" \
		-H "Content-Type: application/json" \
		-d ./docs/swagger.json \
		-o ./docs/swagger3.json

.PHONY: convert-swagger

air: swagger
	cd cmd && air -d

server: swagger
	go run cmd/main.go

docker-build:
	docker build -t ${DOCKER_IMAGE_NAME}:${DOCKER_TAG} .

docker-build-push:
	docker buildx create --use --name multi-arch-builder || true
	docker buildx build --platform linux/amd64,linux/arm64 \
		--build-arg TARGETOS=linux \
		--build-arg TARGETARCH=amd64 \
		-t ${DOCKER_IMAGE_NAME}:${DOCKER_TAG} \
		--push .

docker-push: docker-build-push
	docker push francosae/hitbox-platform-api:${DOCKER_TAG}

docker-run: docker-build
	docker run -p 8080:8080 \
      -v `pwd`/pkg/config/envs/prod.env:/app/config/prod.env \
      -v `pwd`/pkg/config/envs/firebase.json:/app/config/firebase.json \
      ${DOCKER_IMAGE_NAME}:${DOCKER_TAG}

docker-debug:
	docker build --progress=plain -t ${DOCKER_IMAGE_NAME}:${DOCKER_TAG} .
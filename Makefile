NAME := raggs
MAINTAINER := cwlms
VERSION := $(shell grep "const version =" main.go | cut -d\" -f2)

.PHONY: *

default: run

run: build
	docker run -p 3000:3000 ${MAINTAINER}/${NAME}

build: vet
	@echo Building Binary and Container
	@go build -o ${NAME}
	@docker build -t ${MAINTAINER}/${NAME} .

vet:
	@echo Formatting Code
	@go fmt ./...
	@echo Vetting Code
	@go vet .

push: build
	docker tag ${MAINTAINER}/${NAME}:latest ${MAINTAINER}/${NAME}:${VERSION}
	docker push ${MAINTAINER}/${NAME}:latest
	docker push ${MAINTAINER}/${NAME}:${VERSION}

tag:
	git tag v${VERSION}
	git push origin --tags
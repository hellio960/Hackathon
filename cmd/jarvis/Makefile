gen:
	goctl api go -api jarvis.api -dir . -remote git@github.com:qbox/jarvis-template.git -branch main

format:
	#goctl api format --dir ./api -declare
	bash api/formatapi.sh

build-backend:
	go build -o jarvis jarvis.go

build-web:
	cd ../../web && npm install && npm run build

build: build-backend build-web

run:
	go run jarvis.go -f ./etc/jarvis.yaml

web:
	cd ../../web && npm install && npm run build

all:
	make gen && make format && make build

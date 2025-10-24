gen:
	goctl api go -api jarvis.api -dir . -remote git@github.com:qbox/jarvis-template.git -branch main

format:
	#goctl api format --dir ./api -declare
	bash api/formatapi.sh

build:
	go build -o jarvis jarvis.go

run:
	go run jarvis.go -f ./etc/jarvis.yaml

web:
	cd ../../web && npm install && npm run build

all:
	make gen && make format && make build

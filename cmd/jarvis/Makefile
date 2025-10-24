gen:
	goctl api go -api jarvis.api -dir . -remote git@github.com:qbox/jarvis-template.git -branch main

format:
	#goctl api format --dir ./api -declare
	bash api/formatapi.sh

run:
	go run jarvis.go -f ./etc/jarvis-api.yaml

all:
	make gen && make format

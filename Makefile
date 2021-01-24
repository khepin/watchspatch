build:
	pkger -include /assets
	go build

install: build
	go install

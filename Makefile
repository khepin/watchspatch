build:
	@ command -v pkger || go get github.com/markbates/pkger/cmd/pkger
	pkger -include /assets
	go build

install: build
	go install

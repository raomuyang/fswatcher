.PHONY: vendor
vendor:
	@echo "Prepare environment via govendor"
	govendor init
	govendor add +e


.PHONY: prepare
prepare:
	@echo "Prepare environment via go get"
	go get -t ./...

test: prepare
	@echo "Test project"
	go test -race ./...
	go vet ./...

.PHONY: compile
compile: prepare
	@echo "Compile all"
	mkdir -p target
	GOOS=windows GOARCH=amd64 go build  -o target/iusync-win.exe iusync/iusync.go
	GOOS=linux GOARCH=amd64 go build  -o target/iusync-linux iusync/iusync.go
	GOOS=darwin GOARCH=amd64 go build  -o target/iusync-mac iusync/iusync.go
	chmod a+x target/*

.PHONY: clean
clean:
	rm -r target

.PHONY: install
install: prepare
	go install github.com/raomuyang/fswatcher/iusync


all:
	@go install -tags=embed ../vm/jsvm/main
	go build -tags=embed -o ./bin/jsvm ../vm/jsvm/main
	@go install -tags=embed ../vm/luavm/main
	go build -tags=embed -o ./bin/luavm ../vm/luavm/main
	@go install -tags=embed ../cmd/lcnd/
	go build -tags=embed -o ./bin/lcnd ../cmd/lcnd/

tests:
	@go install -tags=embed ../tests/client
	go build -tags=embed -o ./bin/client ../tests/client
	@go install -tags=embed ../tests/contract
	go build -tags=embed -o ./bin/contract ../tests/contract
	@go install -tags=embed ../tests/rpc
	go build -tags=embed -o ./bin/rpc ../tests/rpc

run:
	@./start.sh

stop:
	@./stop.sh

clear:
	@rm -rf ./datadir

.PHONY: all tests run stop clear

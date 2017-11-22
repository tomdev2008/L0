
all:
	go build -tags=embed -o ./bin/jsvm ../vm/jsvm/main
	go build -tags=embed -o ./bin/luavm ../vm/luavm/main
	go build -tags=embed -o ./bin/lcnd ../cmd/lcnd/

tests:
	go build -tags=embed -o ./bin/client ../tests/client
	go build -tags=embed -o ./bin/contract ../tests/contract
	go build -tags=embed -o ./bin/rpc ../tests/rpc

.PHONY: all tests

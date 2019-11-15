build: 
	go build
testunit: 
	go test
testrace: 
	go test -race
test: testunit testrace
all: build test

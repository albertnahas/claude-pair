BINARY := claude-pair
VERSION := 0.1.0

.PHONY: build install clean doctor

build:
	go build -ldflags "-s -w" -o $(BINARY) ./cmd/claude-pair/

install: build
	cp $(BINARY) /usr/local/bin/$(BINARY)

clean:
	rm -f $(BINARY)

doctor: build
	./$(BINARY) doctor

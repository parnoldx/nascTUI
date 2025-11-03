.PHONY: build clean install run test demo live

# Build the C wrapper object
src/calc_wrapper.o: src/calc_wrapper.cpp
	g++ -c -std=c++11 `pkg-config --cflags libqalculate` src/calc_wrapper.cpp -o src/calc_wrapper.o

# Build the main binary
build: src/calc_wrapper.o
	cd src && go build -ldflags "-X main.version=$(shell git describe --tags --abbrev=0 2>/dev/null || echo dev)" -o ../nasc

# Clean build artifacts
clean:
	rm -f src/calc_wrapper.o nasc

# Install system dependencies (Arch Linux)
install-deps:
	sudo pacman -S go libqalculate pkgconf gcc

# Install VHS dependencies (Arch Linux)
install-vhs-deps:
	sudo pacman -S vhs nss atk gtk3 libx11 libxcomposite libxrandr libxdamage libdrm mesa alsa-lib

# Run the application
run: build
	./nasc

# Run tests
test: src/calc_wrapper.o
	cd src && go test -v

# Create demo GIF (requires VHS dependencies)
demo:
	vhs src/demo.tape

# Live reload during development (requires watchexec)
live:
	watchexec -r -e go --wrap-process session --watch src -- "cd src && go build -o ../nasc && cd .. && ./nasc"

# Default target
all: build

# Make 'build' the default when running just 'make'
.DEFAULT_GOAL := build
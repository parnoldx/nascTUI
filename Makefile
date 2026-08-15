.PHONY: all build clean install run test demo live install-deps install-vhs-deps

PREFIX ?= /usr/local
VERSION ?= $(shell git describe --tags --abbrev=0 2>/dev/null || echo dev)

# cgo compiles src/calc_wrapper.cpp automatically
build:
	cd src && go build -ldflags "-X main.version=$(VERSION)" -o ../nasc

all: build

clean:
	rm -f src/calc_wrapper.o nasc

install: build
	install -d "$(DESTDIR)$(PREFIX)/bin"
	install -m 755 nasc "$(DESTDIR)$(PREFIX)/bin/nasc"

# Arch Linux and derivatives (Omarchy, Manjaro, EndeavourOS, ...)
install-deps:
	if command -v omarchy >/dev/null 2>&1 && command -v omarchy-pkg-add >/dev/null 2>&1; then \
		omarchy pkg add go libqalculate pkgconf gcc git; \
	else \
		sudo pacman -S --needed go libqalculate pkgconf gcc git; \
	fi

install-vhs-deps:
	sudo pacman -S --needed vhs nss atk gtk3 libx11 libxcomposite libxrandr libxdamage libdrm mesa alsa-lib

run: build
	./nasc

test:
	cd src && go test -v

demo:
	vhs src/demo.tape

live:
	watchexec -r -e go,cpp --wrap-process session --watch src -- "cd src && go build -o ../nasc && cd .. && ./nasc"

.DEFAULT_GOAL := build

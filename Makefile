SHELL = /bin/bash

ifeq ($(PREFIX),)
    PREFIX := /usr/local
endif
ifeq ($(BINDIR),)
    BINDIR := $(PREFIX)/bin
endif
ifeq ($(SYSCONFDIR),)
    SYSCONFDIR := $(PREFIX)/etc
endif
ifeq ($(GO),)
    GO := $(shell type -a -P go | head -n 1)
endif

build:
	mkdir -p build
	cd src/bpm-convert; $(GO) build -ldflags "-w" -o ../../build/bpm-convert git.enumerated.dev/bubble-package-manager/bpm-utils/src/bpm-convert
	cd src/bpm-package; $(GO) build -ldflags "-w" -o ../../build/bpm-package git.enumerated.dev/bubble-package-manager/bpm-utils/src/bpm-package
	cd src/bpm-repo; $(GO) build -ldflags "-w" -o ../../build/bpm-repo git.enumerated.dev/bubble-package-manager/bpm-utils/src/bpm-repo
	cd src/bpm-setup; $(GO) build -ldflags "-w" -o ../../build/bpm-setup git.enumerated.dev/bubble-package-manager/bpm-utils/src/bpm-setup

install: build config/
	# Create directories
	install -dm755 $(DESTDIR)$(BINDIR)
	install -dm755 $(DESTDIR)$(SYSCONFDIR)
	# Install binaries
	install -Dm755 build/bpm-* -t $(DESTDIR)$(BINDIR)/
	# Install config files
	install -Dm644 config/* -t $(DESTDIR)$(SYSCONFDIR)/bpm-utils/

clean:
	rm -r build/

.PHONY: build
# Installation paths
PREFIX ?= /usr/local
BINDIR ?= $(PREFIX)/bin
SYSCONFDIR := $(PREFIX)/etc

# Compilers and tools
GO ?= go

build:
	mkdir -p build
	cd src/bpm-convert; $(GO) build -ldflags "-w" -o ../../build/bpm-convert git.enumerated.dev/bubble-package-manager/bpm-utils/src/bpm-convert
	cd src/bpm-package; $(GO) build -ldflags "-w" -o ../../build/bpm-package git.enumerated.dev/bubble-package-manager/bpm-utils/src/bpm-package
	cd src/bpm-repo; $(GO) build -ldflags "-w" -o ../../build/bpm-repo git.enumerated.dev/bubble-package-manager/bpm-utils/src/bpm-repo
	cd src/bpm-setup; $(GO) build -ldflags "-w" -o ../../build/bpm-setup git.enumerated.dev/bubble-package-manager/bpm-utils/src/bpm-setup

install:
	# Create directory
	install -dm755 $(DESTDIR)$(BINDIR)
	# Install binary
	install -Dm755 build/bpm-* -t $(DESTDIR)$(BINDIR)/

install-config:
	# Create directory
	install -dm755 $(DESTDIR)$(SYSCONFDIR)
	# Install files
	install -dm755 $(DESTDIR)$(SYSCONFDIR)/bpm-utils/
	cp -r config/* -t $(DESTDIR)$(SYSCONFDIR)/bpm-utils/

uninstall:
	-rm -f $(DESTDIR)$(BINDIR)/bpm-{convert,package,repo-setup}
	-rm -rf $(DESTDIR)$(SYSCONFDIR)/bpm-utils/

clean:
	rm -r build/

.PHONY: build install install-config uninstall clean

GO ?= go
RM ?= rm
SCDOC ?= scdoc
GOFLAGS ?=
PREFIX ?= /usr/local
BINDIR ?= bin
MANDIR ?= share/man

all: rttop doc/rttop.1

rttop:
	$(GO) build $(GOFLAGS) ./cmd/rttop
rttop-server:
	$(GO) build $(GOFLAGS) ./cmd/rttop-server
doc/rttop.1: doc/rttop.1.scd
	$(SCDOC) <doc/rttop.1.scd >doc/rttop.1

clean:
	$(RM) -f rttop rttopctl doc/rttop.1
install:
	mkdir -p $(DESTDIR)$(PREFIX)/$(BINDIR)
	mkdir -p $(DESTDIR)$(PREFIX)/$(MANDIR)/man1
	cp -f rttop rttopctl $(DESTDIR)$(PREFIX)/$(BINDIR)
	cp -f doc/rttop.1 $(DESTDIR)$(PREFIX)/$(MANDIR)/man1

.PHONY: rttop rttop-server clean install

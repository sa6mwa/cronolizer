NAME = cronolize
MODULE = github.com/sa6mwa/cronolizer
VERSION = $(shell git describe --tags --abbrev=0 2>/dev/null || echo 0)
DESTDIR = /usr/local/bin
#GOOS = $(shell uname -s | tr '[:upper:]' '[:lower:]')
SRC = $(MODULE)/cmd/$(NAME)
GO = CGO_ENABLED=0 go
build = $(GO) build -v -ldflags '-s -X main.version=$(VERSION)'
crossCompile = GOOS=$(1) GOARCH=$(2) $(build) -o $(NAME)-$(1)-$(2) $(SRC)
armCompile = GOOS=$(1) GOARCH=arm GOARM=$(2) $(build) -o $(NAME)-$(1)-arm$(2) $(SRC)

.PHONY: all clean build dependencies test amd64 arm64 386 arm% %bsd darwin linux

all: build

release: clean dependencies test linux darwin freebsd netbsd openbsd
	tar --owner=0 --group=0 -czf $(NAME)-$(VERSION).tar.gz --transform 's|^|$(NAME)-$(VERSION)/|' go.* LICENSE Makefile README.md cmd/ $(NAME)-*-*
	sha1sum $(NAME)-*-* > $(NAME)-$(VERSION).sha1sum

install:
	@if [ `id -u` -ne 0 ]; then echo "You may need to sudo to install $(NAME)." ; fi
	for f in $(NAME) $(NAME)-*-* ; do if test -x $$f ; then install $$f $(DESTDIR)/$$f ; fi ; done

clean:
	for f in $(NAME) $(NAME)-*-* ; do if test -x $$f ; then rm -f $$f ; fi; done
	rm -f $(NAME)-$(VERSION).tar.gz $(NAME)-$(VERSION).sha1sum

build: dependencies test $(NAME)

dependencies:
	$(GO) get -v -d ./...

test:
	$(GO) test -cover ./...

go.mod:
	go mod init $(MODULE)
	go mod tidy

linux: amd64 arm64 arm6 arm7 386

amd64: $(NAME)-linux-amd64

arm64: $(NAME)-linux-arm64

386: $(NAME)-linux-386

arm%:
	$(MAKE) $(NAME)-linux-arm6 $(NAME)-linux-$@

%bsd:
	$(MAKE) $(NAME)-$@-amd64 $(NAME)-$@-arm64

darwin: $(NAME)-darwin-amd64 $(NAME)-darwin-arm64

$(NAME):
	$(build) -o $(NAME) $(SRC)

$(NAME)-linux-amd64:
	$(call crossCompile,linux,amd64)

$(NAME)-linux-arm64:
	$(call crossCompile,linux,arm64)

$(NAME)-linux-arm6:
	$(call armCompile,linux,6)

$(NAME)-linux-arm7:
	$(call armCompile,linux,7)

$(NAME)-linux-386:
	$(call crossCompile,linux,386)

$(NAME)-darwin-amd64:
	$(call crossCompile,darwin,amd64)

$(NAME)-darwin-arm64:
	$(call crossCompile,darwin,amd64)

$(NAME)-freebsd-amd64:
	$(call crossCompile,freebsd,amd64)

$(NAME)-freebsd-arm64:
	$(call crossCompile,freebsd,arm64)

$(NAME)-netbsd-amd64:
	$(call crossCompile,netbsd,amd64)

$(NAME)-netbsd-arm64:
	$(call crossCompile,netbsd,arm64)

$(NAME)-openbsd-amd64:
	$(call crossCompile,openbsd,amd64)

$(NAME)-openbsd-arm64:
	$(call crossCompile,openbsd,arm64)

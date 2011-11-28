include $(GOROOT)/src/Make.inc

TARG=github.com/rcrowley/goagain
GOFILES=\
	goagain.go\

include $(GOROOT)/src/Make.pkg

all: uninstall clean install
	make -C cmd/goagain-example uninstall clean install

uninstall:
	rm -f $(GOROOT)/pkg/$(GOOS)_$(GOARCH)/$(TARG).a
	rm -rf $(GOROOT)/src/pkg/$(TARG)
	make -C cmd/goagain-example uninstall

.PHONY: all uninstall

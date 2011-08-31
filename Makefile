include $(GOROOT)/src/Make.inc

TARG=goagain
GOFILES=goagain.go

include $(GOROOT)/src/Make.pkg

all: uninstall clean install
	make -C example uninstall clean install

uninstall:
	rm -f $(GOROOT)/pkg/$(GOOS)_$(GOARCH)/$(TARG).a

.PHONY: uninstall

include $(GOROOT)/src/Make.inc

TARG=goagain
GOFILES=goagain.go

include $(GOROOT)/src/Make.pkg

uninstall:
	rm -f $(GOROOT)/pkg/$(GOOS)_$(GOARCH)/$(TARG).a

.PHONY: uninstall

CMDS=$(shell find src/cmd -mindepth 1 -maxdepth 1 -type d)
PKGS=$(shell find src/pkg -mindepth 1 -maxdepth 1 -type d)

all: $(PKGS)

example: $(CMDS)

$(CMDS) $(PKGS)::
	#make -C $@ install
	make -C $@ uninstall clean install

.PHONY: all

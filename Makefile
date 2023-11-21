.PHONY: build install

BINARY=sshago
INSTALL_PATH=~/bin

build:
	CGO_ENABLED=0 go build -trimpath -ldflags "-s -w" -o $(BINARY)

install: build
	install -m 0755 $(BINARY) $(INSTALL_PATH)
	install -m 0644 init/sshago.service ~/.config/systemd/user/sshago.service
	systemctl --user daemon-reload

enable:
	systemctl --user enable --now sshago.service

uninstall:
	rm -f $(INSTALL_PATH)/$(BINARY)
	rm -f ~/.config/systemd/user/sshago.service
	systemctl --user daemon-reload


.PHONY: build gen cli

COMPONENTS = queue-service downloader cli
TOOLCHAINS = darwin/arm64 freebsd/amd64 linux/amd64
TARGETS = gen build

$(TARGETS):
	for tc in $(TOOLCHAINS); do \
		for dir in $(COMPONENTS); do \
  			GOOS=$$(echo $$tc | cut -d/ -f1) GOARCH=$$(echo $$tc | cut -d/ -f2) $(MAKE) -C $$dir $@; \
  		done \
  	done

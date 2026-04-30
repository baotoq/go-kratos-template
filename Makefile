.PHONY: api
# generate api
api:
	find app -mindepth 1 -maxdepth 1 -type d -print | xargs -L 1 bash -c 'cd "$$0" && pwd && $(MAKE) api'

.PHONY: wire
# generate wire
wire:
	find app -mindepth 1 -maxdepth 1 -type d -print | xargs -L 1 bash -c 'cd "$$0" && pwd && $(MAKE) wire'

.PHONY: proto
# generate proto
proto:
	find app -mindepth 1 -maxdepth 1 -type d -print | xargs -L 1 bash -c 'cd "$$0" && pwd && $(MAKE) proto'

.PHONY: all
# generate api
all:
	find app -mindepth 1 -maxdepth 1 -type d -print | xargs -L 1 bash -c 'cd "$$0" && pwd && $(MAKE) all'

.PHONY: dev
# start dev environment with tilt
dev:
	tilt up -- --continue

.PHONY: debug
# start dev environment, wait for debugger to attach
debug:
	tilt up
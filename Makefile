SHELL := /bin/bash

.PHONY: all
all:
	${MAKE} -C tgapi
	${MAKE} -C tgrun
	${MAKE} -C web

.PHONY: run
run: 
	${MAKE} -C tgapi build
	${MAKE} -C tgrun build
	@skaffold run --default-repo ttl.sh/dev-testgrid

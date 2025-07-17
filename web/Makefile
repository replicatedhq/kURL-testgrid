SHELL := /bin/bash
PROJECT_NAME ?= tgweb

.PHONY: all
all: deps lint build-okteto

.PHONY: deps
deps:
	npm install

depcheck:
	# npm install depcheck -g
	depcheck . --specials=webpack,babel,bin,eslint

.PHONY: lint
lint:
	npx eslint ./src --ext .js,.jsx,.ts,.tsx

.PHONY: serve
serve: export NODE_ENV = development
serve:
	npm start

.PHONY: serve-okteto
serve-okteto: export NODE_ENV = okteto
serve-okteto:
	npm start

.PHONY: build
build:
	@echo "nothing to do"

.PHONY: run
run: serve

.PHONY: build-okteto
build-okteto: deps
build-okteto:
	node \
		--max_old_space_size=6144 \
		./node_modules/webpack/bin/webpack.js \
		--config webpack.config.js \
		--env okteto \
		--mode development
	cp _redirects dist

.PHONY: build-staging
build-staging: deps
build-staging:
	node \
		--max_old_space_size=6144 \
		./node_modules/webpack/bin/webpack.js \
		--config webpack.config.js \
		--env staging \
		--mode production
	cp _redirects dist-staging

.PHONY: build-production
build-production: deps
build-production:
	node \
		--max_old_space_size=6144 \
		./node_modules/webpack/bin/webpack.js \
		--config webpack.config.js \
		--env production \
		--mode production
	cp _redirects dist-prod
GO ?= go
GOFMT ?= gofmt
PYTHON ?= python3.13

.PHONY: fmt test test-go test-python dev gateway baseline-orchestrator decision-engine migrate-mysql secret-scan install-hooks

fmt:
	$(GOFMT) -w $$(find . -name '*.go')

test: test-go test-python

test-go:
	$(GO) test ./...

test-python:
	PYTHONPATH=apps/decision-engine $(PYTHON) -m unittest discover -s apps/decision-engine/tests -p 'test_*.py'

dev:
	$(GO) run ./cmd/dev-server

gateway:
	$(GO) run ./cmd/investigation-gateway

baseline-orchestrator:
	$(GO) run ./cmd/baseline-orchestrator

decision-engine:
	cd apps/decision-engine && $(PYTHON) -m decision_engine

migrate-mysql:
	scripts/mysql-migrate.sh

secret-scan:
	$(PYTHON) scripts/secret-scan.py --mode all

install-hooks:
	scripts/install-git-hooks.sh

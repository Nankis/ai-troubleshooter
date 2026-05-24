GO ?= go
GOFMT ?= gofmt
PYTHON ?= $(shell if [ -x .venv/bin/python ]; then echo .venv/bin/python; else echo python3.13; fi)

.PHONY: fmt test test-go test-python dev agent-platform gateway decision-engine migrate-mysql secret-scan install-hooks

fmt:
	$(GOFMT) -w $$(find . -name '*.go')

test: test-go test-python

test-go:
	$(GO) test ./...

test-python:
	PYTHONPATH=apps/decision-engine $(PYTHON) -m unittest discover -s apps/decision-engine/tests -p 'test_*.py'
	PYTHONPATH=apps/agent-platform:apps/decision-engine $(PYTHON) -m unittest discover -s apps/agent-platform/tests -p 'test_*.py'
	$(PYTHON) -m unittest discover -s tests -p 'test_*.py'

dev:
	PYTHONPATH=apps/agent-platform:apps/decision-engine $(PYTHON) -m agent_platform

agent-platform:
	PYTHONPATH=apps/agent-platform:apps/decision-engine $(PYTHON) -m agent_platform

gateway:
	$(GO) run ./cmd/investigation-gateway

decision-engine:
	cd apps/decision-engine && $(PYTHON) -m decision_engine

migrate-mysql:
	scripts/mysql-migrate.sh

secret-scan:
	$(PYTHON) scripts/secret-scan.py --mode all

install-hooks:
	scripts/install-git-hooks.sh

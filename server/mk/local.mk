# =======================
# Local
# =======================

build:
	go build -ldflags="-X main.version=0.0.1" ./cmd/reearth

clean:
	go clean -modcache
	go clean -cache
	go clean -testcache

deep-copy:
	(cd pkg/property && deep-copy --type Initializer --pointer-receiver -o initializer_gen.go .)

dev: dev-install
	air

dev-install:
	@for bin in air stringer schematyper deep-copy; do \
		if ! which $$bin >/dev/null 2>&1; then \
			echo "$$bin is not installed. Installing..."; \
			case $$bin in \
				air) go install github.com/air-verse/air@v1.61.5 ;; \
				stringer) go install golang.org/x/tools/cmd/stringer@v0.39.0 ;; \
				schematyper) go install github.com/idubinskiy/schematyper ;; \
				deep-copy) go install github.com/globusdigital/deep-copy@v0.5.5-0.20251122193020-7cda106f7b4b ;; \
			esac; \
		else \
			echo "$$bin is already installed."; \
		fi; \
	done

e2e:
	go test -v ./e2e/...

error-msg:
	go generate ./pkg/i18n/...

generate: dev-install
	go generate ./...

gql:
	go generate ./internal/adapter/gql/gqldataloader
	go generate ./internal/adapter/gql

lint:
	golangci-lint run --fix

migrate:
	@echo "==== Running database migration ===="
	RUN_MIGRATION=true REEARTH_DB=${REEARTH_DB} REEARTH_DB_VIS=${REEARTH_DB_VIS} go run ./cmd/reearth
	@echo "✓ Migration complete!"

migrate-with-key:
	@if [ -z "$(MIGRATION_KEY)" ]; then \
		echo "Error: MIGRATION_KEY is not set."; \
		echo "Usage: make migrate-with-key MIGRATION_KEY=<key>"; \
		exit 1; \
	fi
	@echo "==== Running database migration with MIGRATION_KEY=$(MIGRATION_KEY) ===="
	RUN_MIGRATION=true MIGRATION_KEY=$(MIGRATION_KEY) REEARTH_DB=${REEARTH_DB} REEARTH_DB_VIS=${REEARTH_DB_VIS} go run ./cmd/reearth
	@echo "✓ Migration complete!"

run-standalone:
	go run -ldflags="-X main.version=0.0.1" ./cmd/reearth

run-app:
	@if [ ! -f .env ]; then \
		echo "Creating .env from .env.example..."; \
		cp .env.example .env; \
	fi
	make d-run-accounts
	make mockuser
	make d-up-gcs
	make gcs-bucket
	go run -ldflags="-X main.version=0.0.1" ./cmd/reearth

run-clean-start: clean run-app

schematyper:
	go run $(SCHEMATYPER) -o $(MANIFEST_DIR)/schema_translation.go --package manifest --prefix Translation ./schemas/plugin_manifest_translation.json
	go run $(SCHEMATYPER) -o $(MANIFEST_DIR)/schema_gen.go --package manifest ./schemas/plugin_manifest.json

test:
	@if [ -d tmp/mongo ] && [ ! -r tmp/mongo ]; then \
		echo >&2 "tmp/mongo under server/ is not readable (legacy Docker bind mount as root). Remove it:"; \
		echo >&2 "  sudo rm -rf tmp/mongo tmp/gcs"; \
		exit 1; \
	fi
	@if [ -d tmp/gcs ] && [ ! -r tmp/gcs ]; then \
		echo >&2 "tmp/gcs under server/ is not readable. Remove it:"; \
		echo >&2 "  sudo rm -rf tmp/mongo tmp/gcs"; \
		exit 1; \
	fi
	REEARTH_DB=${REEARTH_DB} go test $(if $(TEST_P),-p $(TEST_P),) ${TEST_DIR} -run ${TARGET_TEST}

# Full package list except e2e (same REEARTH_DB). Use when local Mongo drops under full ./...
# (e.g. VM + Docker): make test-no-e2e TEST_P=2
test-no-e2e:
	@if [ -d tmp/mongo ] && [ ! -r tmp/mongo ]; then \
		echo >&2 "tmp/mongo under server/ is not readable (legacy Docker bind mount as root). Remove it:"; \
		echo >&2 "  sudo rm -rf tmp/mongo tmp/gcs"; \
		exit 1; \
	fi
	@if [ -d tmp/gcs ] && [ ! -r tmp/gcs ]; then \
		echo >&2 "tmp/gcs under server/ is not readable. Remove it:"; \
		echo >&2 "  sudo rm -rf tmp/mongo tmp/gcs"; \
		exit 1; \
	fi
	REEARTH_DB=${REEARTH_DB} go test $(if $(TEST_P),-p $(TEST_P),) $$(go list ./... | grep -v '/e2e$$') -run ${TARGET_TEST}

test-debug:
	go test -v -timeout 10s ${TEST_DIR} | tee test.log

.PHONY: build clean deep-copy dev dev-install e2e error-msg generate gql lint migrate migrate-with-key run-app run-clean-start run-standalone schematyper test test-no-e2e test-debug

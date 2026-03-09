# =====================================================
# TOOL BINARIES
# =====================================================

SQLC ?= sqlc
OAPI ?= oapi-codegen
GO   ?= go

# =====================================================
# PATHS
# =====================================================

OPENAPI_FILE := api/openapi.yaml
OPENAPI_OUT  := pkg/api/api.gen.go
SQLC_CONFIG  := sql/sqlc.yaml


# =====================================================
# DEFAULT TARGET
# =====================================================

.PHONY: all
all: generate


# =====================================================
# GENERATE
# =====================================================

.PHONY: generate
generate: generate-sqlc generate-oapi

# --- SQLC ---
.PHONY: generate-sqlc
generate-sqlc:
	@echo ">> Generating SQLC code..."
	$(SQLC) generate -f $(SQLC_CONFIG)

# --- OAPI ---
.PHONY: generate-oapi
generate-oapi:
	@echo ">> Generating OpenAPI code..."
	$(OAPI) \
		-generate types,std-http-server,strict-server,spec \
		-package api \
		-o $(OPENAPI_OUT) \
		$(OPENAPI_FILE)


# =====================================================
# GO MODULE
# =====================================================

.PHONY: tidy
tidy:
	$(GO) mod tidy


# =====================================================
# CLEAN GENERATED FILES
# =====================================================

.PHONY: clean
clean:
	rm -f pkg/api/*.gen.go


# =====================================================
# INSTALL DEV TOOLS
# =====================================================

.PHONY: install-tools
install-tools:
	@echo ">> Installing sqlc..."
	$(GO) install github.com/sqlc-dev/sqlc/cmd/sqlc@latest
	@echo ">> Installing oapi-codegen..."
	$(GO) install github.com/deepmap/oapi-codegen/v2/cmd/oapi-codegen@latest

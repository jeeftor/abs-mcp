# syntax=docker/dockerfile:1

ARG BUILDER_IMAGE=golang:1.26-bookworm

FROM ${BUILDER_IMAGE} AS build

ARG TARGETOS=linux
ARG TARGETARCH=amd64
ARG INSTALL_MITRE_CERTS=false

WORKDIR /src
ENV GOCACHE=/tmp/go-build

# Corporate-network builds can opt in to the MITRE PKI install without making
# public GitHub builds depend on an internal certificate endpoint.
RUN if [ "$INSTALL_MITRE_CERTS" = "true" ]; then \
		apt-get update && \
		apt-get install -y --no-install-recommends ca-certificates curl && \
		curl -ksSL https://gitlab.mitre.org/mitre-scripts/mitre-pki/raw/master/os_scripts/install_certs.sh | sh && \
		rm -rf /var/lib/apt/lists/*; \
	fi

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH \
	go build -trimpath -ldflags="-s -w" -o /out/abs-mcp ./cmd/abs-mcp

FROM scratch

COPY --from=build /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
COPY --from=build /out/abs-mcp /abs-mcp
COPY --from=build /src/docs/api-inventory/generated/abs-api-inventory.json /docs/api-inventory/generated/abs-api-inventory.json

USER 65532:65532
ENTRYPOINT ["/abs-mcp"]

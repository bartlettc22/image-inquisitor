ARG GO_VERSION=latest
ARG TRIVY_VERSION=latest
ARG HELM_VERSION=3.18.6
ARG VERSION=dev

FROM aquasec/trivy:${TRIVY_VERSION} AS trivy

FROM golang:${GO_VERSION}-alpine AS builder

RUN apk --no-cache add ca-certificates

# helm renders Argo CD apps' charts
ARG HELM_VERSION
RUN cd /tmp \
    && wget -q https://get.helm.sh/helm-v${HELM_VERSION}-linux-amd64.tar.gz \
    && wget -q https://get.helm.sh/helm-v${HELM_VERSION}-linux-amd64.tar.gz.sha256sum \
    && sha256sum -c helm-v${HELM_VERSION}-linux-amd64.tar.gz.sha256sum \
    && tar -xzf helm-v${HELM_VERSION}-linux-amd64.tar.gz linux-amd64/helm \
    && mv linux-amd64/helm /helm

WORKDIR /src
COPY go.mod go.sum /src/
RUN --mount=type=cache,target=/go/pkg/mod go mod download

COPY . /src

RUN --mount=type=cache,target=/root/.cache/go-build \
    --mount=type=cache,target=/go/pkg/mod \
    CGO_ENABLED=0 \
    GOOS=linux \
    GOARCH=amd64 \
    go build \
    -ldflags "-s -w -extldflags \"-static\"  -X 'main.version=${VERSION}'" \
    -o /imginq \
    ./cmd

# helm and trivy need a writable /tmp; the SQLite database, caches and SBOMs go in /var/lib/imginq
RUN mkdir -p /data/tmp /data/var/lib/imginq

FROM scratch

# Links the GHCR package to the repo
LABEL org.opencontainers.image.source=https://github.com/bartlettc22/image-inquisitor

ARG UID=1001

WORKDIR /

COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

COPY --from=builder --chown=${UID} --chmod=755 /data /
COPY --from=trivy /usr/local/bin/trivy /usr/local/bin/trivy
COPY --from=builder /helm /usr/local/bin/helm
COPY --from=builder /imginq /usr/local/bin/imginq

# This needs to be the UID (vs. the user name) because Kubernetes will check to ensure this is not a root user
# It can only do this if the UID is used so it can ensure that it is != 0 (root)
USER ${UID}

ENTRYPOINT ["imginq"]
CMD ["serve", "--config", "/etc/imginq/config.yaml"]

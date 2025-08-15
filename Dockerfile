ARG GO_VERSION=latest
ARG TRIVY_VERSION=latest
ARG VERSION=latest

FROM aquasec/trivy:${TRIVY_VERSION} AS trivy

FROM golang:${GO_VERSION}-alpine AS builder

RUN apk --no-cache add ca-certificates

WORKDIR /src
COPY . /src

RUN --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 \
    GOOS=linux \
    GOARCH=amd64 \
    go build \
    -ldflags "-s -w -extldflags \"-static\"  -X 'main.version=${VERSION}'" \
    -o /imginq \
    ./cmd

RUN mkdir -p /data/tmp
RUN mkdir -p /data/trivy
RUN mkdir -p /data/reports

FROM scratch

ARG UID=1001

WORKDIR /

COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

COPY --from=builder --chown=${UID} --chmod=755 /data /
COPY --from=trivy /usr/local/bin/trivy /usr/local/bin/trivy
COPY --from=builder /imginq /usr/local/bin/imginq

ENV TRIVY_CACHE_DIR=/trivy/cache

# This needs to be the UID (vs. the user name) because Kubernetes will check to ensure this is not a root user
# It can only do this if the UID is used so it can ensure that it is != 0 (root)
USER ${UID}

ENTRYPOINT ["imginq"]

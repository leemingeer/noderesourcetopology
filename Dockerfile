ARG BUILDER_IMAGE
ARG BASE_IMAGE_FULL
ARG BASE_IMAGE_MINIMAL

# Build node feature discovery
FROM ${BUILDER_IMAGE} as builder

# Get (cache) deps in a separate layer
COPY go.mod go.sum /go/topology-updater/

WORKDIR /go/topology-updater

RUN export GOPROXY=https://goproxy.cn,direct && go mod download

# Do actual build
COPY . /go/topology-updater

ARG VERSION
ARG HOSTMOUNT_PREFIX

RUN make install VERSION=$VERSION HOSTMOUNT_PREFIX=$HOSTMOUNT_PREFIX

# Create full variant of the production image
FROM ${BASE_IMAGE_FULL} as full

# Run as unprivileged user
USER 65534:65534

# Use more verbose logging of gRPC
ENV GRPC_GO_LOG_SEVERITY_LEVEL="INFO"

#COPY --from=builder /go/topology-updater/deployment/components/worker-config/nfd-worker.conf.example /etc/kubernetes/topology-updater/nfd-worker.conf
COPY --from=builder /go/bin/* /usr/bin/

# Create minimal variant of the production image
FROM ${BASE_IMAGE_MINIMAL} as minimal

# Run as unprivileged user
USER 65534:65534

# Use more verbose logging of gRPC
ENV GRPC_GO_LOG_SEVERITY_LEVEL="INFO"

#COPY --from=builder /go/topology-updater/deployment/components/worker-config/nfd-worker.conf.example /etc/kubernetes/topology-updater/nfd-worker.conf
COPY --from=builder /go/bin/* /usr/bin/

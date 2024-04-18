ARG BASE_IMAGE

FROM registry.ddbuild.io/images/mirror/golang:1.20 as builder
ARG TARGETARCH
ENV ARCH=$TARGETARCH
COPY . /src
WORKDIR /src
RUN make azuredisk-debug
RUN CGO_ENABLED=0 go install -ldflags "-s -w -extldflags '-static'" github.com/go-delve/delve/cmd/dlv@latest

FROM $BASE_IMAGE
ARG TARGETARCH
COPY --from=builder /src/_output/${TARGETARCH}/azurediskplugin /bin/azurediskplugin
COPY --from=builder /go/bin/dlv /dlv
ENTRYPOINT ["/dlv"]

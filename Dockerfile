ARG BASE_IMAGE

FROM registry.ddbuild.io/images/mirror/golang:1.20 as builder
ARG TARGETARCH
ENV ARCH=$TARGETARCH
COPY . /src
WORKDIR /src
RUN make azuredisk

FROM $BASE_IMAGE
ARG TARGETARCH
COPY --from=builder /src/_output/${TARGETARCH}/azurediskplugin /bin/azurediskplugin
ENTRYPOINT ["/bin/azurediskplugin"]

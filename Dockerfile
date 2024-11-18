ARG BASE_IMAGE
ARG BUILDER_IMAGE

FROM $BUILDER_IMAGE as builder
ARG TARGETARCH
ENV ARCH=$TARGETARCH
COPY . /src
WORKDIR /src
RUN make azuredisk

FROM $BASE_IMAGE
ARG TARGETARCH
COPY --from=builder /src/_output/${TARGETARCH}/azurediskplugin /bin/azurediskplugin
ENTRYPOINT ["/bin/azurediskplugin"]

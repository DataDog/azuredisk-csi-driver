ARG BASE_IMAGE
ARG BUILDER_IMAGE

FROM $BUILDER_IMAGE as builder
ARG TARGETARCH
ENV ARCH=$TARGETARCH
ENV GOTOOLCHAIN auto
COPY . /src
WORKDIR /src
RUN make azuredisk

FROM $BASE_IMAGE
ARG TARGETARCH
# Install xfsprogs for xfs filesystem support
USER root
RUN clean-apt install xfsprogs
USER dog
COPY --from=builder /src/_output/${TARGETARCH}/azurediskplugin /bin/azurediskplugin
ENTRYPOINT ["/bin/azurediskplugin"]

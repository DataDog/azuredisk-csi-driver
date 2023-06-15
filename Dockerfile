ARG BASE_IMAGE

FROM golang:1.20.5 as builder
COPY . /src
WORKDIR /src
RUN make azuredisk

FROM $BASE_IMAGE
ARG TARGETARCH
COPY --from=builder /src/_output/${TARGETARCH}/azurediskplugin /bin/azurediskplugin
ENTRYPOINT ["/bin/azurediskplugin"]

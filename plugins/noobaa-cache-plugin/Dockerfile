FROM ubuntu:16.04 as base
RUN apt-get update && \
  apt-get install -y \
  git wget gcc make mercurial && \
  rm -rf /var/lib/apt/lists/*

ARG ARCH

ENV ARCH=$ARCH
ENV GO_VERSION=1.14.10

RUN echo $ARCH $GO_VERSION

RUN wget -q https://dl.google.com/go/go$GO_VERSION.linux-$ARCH.tar.gz && \
  tar -xf go$GO_VERSION.linux-$ARCH.tar.gz && \
  rm go$GO_VERSION.linux-$ARCH.tar.gz && \
  mv go /usr/local

ENV GOROOT /usr/local/go
ENV GOPATH /go
ENV PATH=$GOPATH/bin:$GOROOT/bin:$PATH
ENV GOARCH $ARCH
ENV GO111MODULE=on

COPY go.mod .
COPY go.sum .

RUN go mod download

COPY . /noobaa-cache-plugin
WORKDIR /noobaa-cache-plugin
RUN go build -o /noobaa-cache-plugin/build/_output/bin/noobaa-cache-plugin /noobaa-cache-plugin/cmd/manager

FROM registry.access.redhat.com/ubi8/ubi-minimal:latest

ENV OPERATOR=/usr/local/bin/noobaa-cache-plugin \
    USER_UID=1001 \
    USER_NAME=noobaa-cache-plugin

# install operator binary
COPY --from=base /noobaa-cache-plugin/build/_output/bin/noobaa-cache-plugin ${OPERATOR}

COPY build/bin /usr/local/bin
RUN  /usr/local/bin/user_setup

ENTRYPOINT ["/usr/local/bin/entrypoint"]

USER ${USER_UID}
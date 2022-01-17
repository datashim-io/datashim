FROM ubuntu:16.04 as base
RUN apt-get update && \
  apt-get install -y \
  git wget gcc make mercurial && \
  rm -rf /var/lib/apt/lists/*

ARG ARCH

ENV ARCH=$ARCH
#ENV GO_VERSION=1.13.8
ENV GO_VERSION=1.16


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

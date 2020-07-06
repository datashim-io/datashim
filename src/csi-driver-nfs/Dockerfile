FROM ubuntu:16.04 as base
RUN apt-get update && \
  apt-get install -y \
  git wget gcc make mercurial && \
  rm -rf /var/lib/apt/lists/*

ARG ARCH

ENV ARCH=$ARCH
ENV GO_VERSION=1.12.17

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
COPY . /csi-driver-nfs
WORKDIR /csi-driver-nfs
RUN mkdir -p /bin
RUN CGO_ENABLED=0 GOOS=linux go build -a -ldflags '-X main.version=$(REV) -extldflags "-static"' -o /bin/nfsplugin /csi-driver-nfs/cmd/nfsplugin

FROM centos:7

# Copy nfsplugin from build _output directory
COPY --from=base /bin/nfsplugin /nfsplugin

RUN yum -y install nfs-utils && yum -y install epel-release && yum -y install jq && yum clean all

ENTRYPOINT ["/nfsplugin"]

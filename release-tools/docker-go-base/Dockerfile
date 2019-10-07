FROM ubuntu:18.04
RUN apt-get update && \
  apt-get install -y \
  git wget make mercurial && \
  rm -rf /var/lib/apt/lists/*

ARG ARCH
ARG DOCKER_ARCH

ENV ARCH=$ARCH
ENV DOCKER_ARCH=$DOCKER_ARCH

RUN wget -q https://dl.google.com/go/go1.12.10.linux-$ARCH.tar.gz && \
  tar -xf go1.12.10.linux-$ARCH.tar.gz && \
  rm go1.12.10.linux-$ARCH.tar.gz && \
  mv go /usr/local

ENV GOROOT /usr/local/go
ENV GOPATH /go
ENV PATH=$GOPATH/bin:$GOROOT/bin:$PATH

RUN apt-get update && \
    apt-get -y install apt-transport-https \
    ca-certificates \
    curl \
    gnupg2 \
    software-properties-common && \
    curl -fsSL https://download.docker.com/linux/$(. /etc/os-release; echo "$ID")/gpg > /tmp/dkey; apt-key add /tmp/dkey && \
    add-apt-repository \
        "deb [arch=$DOCKER_ARCH] https://download.docker.com/linux/$(. /etc/os-release; echo "$ID") \
        $(lsb_release -cs) \
        stable" && \
    apt-get update && \
    apt-get -y install docker-ce

RUN go get -d github.com/operator-framework/operator-sdk; exit 0
RUN cd $GOPATH/src/github.com/operator-framework/operator-sdk && \
    git checkout v0.10.0 && \
    make tidy && \
    make install
ENV GO111MODULE=on
FROM golang:1.22-bookworm as base
RUN apt-get update && \
  apt-get install -y \
  git wget gcc make mercurial && \
  rm -rf /var/lib/apt/lists/*

ENV GO111MODULE=on

COPY go.mod .
COPY go.sum .

RUN go mod download
COPY . /csi-driver-nfs
WORKDIR /csi-driver-nfs
RUN mkdir -p /bin
RUN CGO_ENABLED=0 GOOS=linux go build -a -ldflags '-X main.version=$(REV) -extldflags "-static"' -o /bin/nfsplugin /csi-driver-nfs/cmd/nfsplugin

FROM mirror.gcr.io/debian:bookworm-slim

# Copy nfsplugin from build _output directory
COPY --from=base /bin/nfsplugin /nfsplugin

RUN apt-get update && apt-get -y install nfs-common && apt-get -y install jq && apt-get clean all

ENTRYPOINT ["/nfsplugin"]

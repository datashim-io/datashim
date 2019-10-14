FROM debian:stretch
LABEL maintainers="Cyrill Troxler <cyrilltroxler@gmail.com>"
LABEL description="csi-s3 slim image"

RUN apt-get update -y && apt-get upgrade -y && apt-get install -y \
libfuse2 gcc sqlite3 libsqlite3-dev \
psmisc procps libcurl3 xfsprogs curl unzip \
libfuse-dev libssl-dev git autotools-dev automake libc6-dev build-essential libcurl4-openssl-dev libxml2-dev pkg-config && \
git clone https://github.com/s3fs-fuse/s3fs-fuse.git && cd s3fs-fuse && ./autogen.sh && ./configure && make && make install && \
rm -rf /var/lib/apt/lists/*

# install rclone
ARG RCLONE_VERSION=v1.47.0
RUN cd /tmp \
  && curl -O https://downloads.rclone.org/${RCLONE_VERSION}/rclone-${RCLONE_VERSION}-linux-amd64.zip \
  && unzip /tmp/rclone-${RCLONE_VERSION}-linux-amd64.zip \
  && mv /tmp/rclone-*-linux-amd64/rclone /usr/bin \
  && rm -r /tmp/rclone*

COPY ./_output/s3driver /s3driver
ENTRYPOINT ["/s3driver"]

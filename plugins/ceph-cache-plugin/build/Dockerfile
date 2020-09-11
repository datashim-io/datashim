FROM registry.access.redhat.com/ubi8/ubi-minimal:latest

ENV OPERATOR=/usr/local/bin/ceph-cache-plugin \
    USER_UID=1001 \
    USER_NAME=ceph-cache-plugin

# install operator binary
COPY build/_output/bin/ceph-cache-plugin ${OPERATOR}

COPY build/bin /usr/local/bin
RUN  /usr/local/bin/user_setup

ENTRYPOINT ["/usr/local/bin/entrypoint"]

USER ${USER_UID}

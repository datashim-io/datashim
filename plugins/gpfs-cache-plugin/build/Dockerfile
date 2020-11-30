FROM registry.access.redhat.com/ubi8/ubi-minimal:latest

ENV OPERATOR=/usr/local/bin/gpfs-cache-plugin \
    USER_UID=1001 \
    USER_NAME=gpfs-cache-plugin

# install operator binary
COPY build/_output/bin/gpfs-cache-plugin ${OPERATOR}

COPY build/bin /usr/local/bin
RUN  /usr/local/bin/user_setup

ENTRYPOINT ["/usr/local/bin/entrypoint"]

USER ${USER_UID}

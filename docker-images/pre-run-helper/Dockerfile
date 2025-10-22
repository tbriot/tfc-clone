# bookworm slim
FROM debian:12.7-slim
WORKDIR /usr/local/app

RUN apt update
RUN apt install -y curl

COPY pre-run-helper ./

# Setup an app user so the container doesn't run as the root user
RUN useradd -m app

# Creating directories where docker volumes will be mounted
# in this Dockerfile so that unix permissions can be set as expected.
# Otherwise those directories are owned by root.
RUN mkdir -p /opt/shared-volume
RUN chown -R app:app /opt/shared-volume
USER app
VOLUME ["/opt/shared-volume"]
CMD ["./pre-run-helper"]

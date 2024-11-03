# bookworm slim
FROM debian:12.7-slim
WORKDIR /usr/local/app

RUN apt update
RUN apt install -y curl

COPY run-executor ./

# Setup an app user so the container doesn't run as the root user
RUN useradd -m app

# Creating directories where docker volumes will be mounted
# in this Dockerfile so that unix permissions can be set as expected.
# Otherwise those directories are owned by root.
RUN mkdir -p /opt/shared-volume
RUN chown -R app:app /opt/shared-volume

RUN mkdir -p /opt/tfc-cache
RUN chown -R app:app /opt/tfc-cache

USER app

ENV HOME /home/app
RUN alias ll='ls -al'

# Install TFSwitch
RUN curl -L https://raw.githubusercontent.com/warrensbox/terraform-switcher/master/install.sh | bash -s -- -b $HOME/.bin
COPY .tfswitch.toml $HOME
ENV PATH="$PATH:$HOME/.bin"

WORKDIR /usr/local/app

VOLUME ["/opt/shared-volume"]
VOLUME ["/opt/tfc-cache"]
CMD ["./run-executor"]

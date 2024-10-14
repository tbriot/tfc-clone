# bookworm slim
FROM debian:12.7-slim
WORKDIR /usr/local/app

RUN apt update
RUN apt install -y curl

COPY run-event-handler ./

# Setup an app user so the container doesn't run as the root user
RUN useradd -m app
USER app

ENV HOME /home/app

# Install TFSwitch
RUN curl -L https://raw.githubusercontent.com/warrensbox/terraform-switcher/master/install.sh | bash -s -- -b $HOME/.bin
COPY .tfswitch.toml $HOME
ENV PATH="$PATH:$HOME/.bin"

WORKDIR /usr/local/app
CMD ["./run-event-handler"]

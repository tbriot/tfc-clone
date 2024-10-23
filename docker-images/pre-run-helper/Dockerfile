# bookworm slim
FROM debian:12.7-slim
WORKDIR /usr/local/app

RUN apt update
RUN apt install -y curl

COPY pre-run-helper ./

# Setup an app user so the container doesn't run as the root user
RUN useradd -m app
USER app

CMD ["./pre-run-helper"]

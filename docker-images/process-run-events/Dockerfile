# bookworm slim
FROM debian:12.7-slim
WORKDIR /usr/local/app

COPY run-event-handler ./

# Setup an app user so the container doesn't run as the root user
RUN useradd app
USER app

CMD ["./run-event-handler"]

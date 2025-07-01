FROM debian:bookworm
ARG TARGETARCH

WORKDIR /app

RUN apt-get update && \
    apt-get install -y ca-certificates

COPY tasmota-exporter-$TARGETARCH /usr/bin/tasmota-exporter

CMD ["/usr/bin/tasmota-exporter"]

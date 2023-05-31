# syntax=docker/dockerfile:1

# Use distroless/static:nonroot image for a base.
FROM --platform=linux/amd64 gcr.io/distroless/static@sha256:1b4dbd7d38a0fd4bbaf5216a21a615d07b56747a96d3c650689cbb7fdc412b49 as linux-amd64

FROM --platform=linux/${TARGETARCH} linux-${TARGETARCH}

# Run as nonroot user using numeric ID for compatibllity.
USER 65532

COPY ghx /usr/local/bin/ghx

ENTRYPOINT ["/usr/local/bin/ghx"]

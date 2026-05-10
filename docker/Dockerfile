FROM gcr.io/distroless/static-debian13:nonroot

ARG TARGETPLATFORM

COPY $TARGETPLATFORM/semverkzeug /usr/local/bin/semverkzeug

ENTRYPOINT ["/usr/local/bin/semverkzeug"]

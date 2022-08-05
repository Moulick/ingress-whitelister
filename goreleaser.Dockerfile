FROM gcr.io/distroless/static:nonroot
COPY ingress-whitelister /
ENTRYPOINT ["/ingress-whitelister"]

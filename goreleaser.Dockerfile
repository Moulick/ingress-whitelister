FROM scratch
LABEL org.opencontainers.image.authors=moulickaggarwal
COPY ingress-whitelister /
ENTRYPOINT ["/ingress-whitelister"]

FROM scratch

USER 65532:65532

# uncors binds to loopback by default, which inside a container means nothing
# published with -p can reach it. A container is its own network namespace, so
# binding every interface here is the equivalent of the host-side default.
# There is no TTY either, so the terminal UI is off.
#
# HOME is where the local CA lives; mount a volume there to use https mappings.
ENV HOME=/home/nonroot

COPY uncors /bin/uncors

EXPOSE 80
EXPOSE 443
EXPOSE 3000

ENTRYPOINT ["/bin/uncors", "--listen", "0.0.0.0", "--interactive=false"]

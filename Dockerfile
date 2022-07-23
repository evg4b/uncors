FROM scratch
COPY uncors /bin/uncors
ENTRYPOINT ["/bin/uncors"]
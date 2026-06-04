FROM scratch
USER 65532:65532
COPY uncors /bin/uncors
EXPOSE 80
EXPOSE 443
ENTRYPOINT ["/bin/uncors"]
FROM scratch
COPY uncors /bin/uncors
EXPOSE 80
EXPOSE 443
ENTRYPOINT ["/bin/uncors"]
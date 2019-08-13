FROM alpine
COPY stripe /bin/stripe
ENTRYPOINT ["/bin/stripe"]

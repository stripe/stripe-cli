FROM alpine
RUN apk add ca-certificates
COPY stripe /bin/stripe
ENTRYPOINT ["/bin/stripe"]

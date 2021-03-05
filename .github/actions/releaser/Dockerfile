ARG GOLANG_VERSION=1.16.0
ARG GORELEASER_VERSION=0.158.0

# todo update
ARG GORELEASER_SHA=d629af53ab7493cadb734299590fd12f6ec54256b5f8b0554032b10b2e15e176

# We're not compiling against 1.13.15, this is just to pin against a specific version
FROM dockercore/golang-cross:1.13.15

ARG GORELEASER_DOWNLOAD_FILE=goreleaser_Linux_x86_64.tar.gz
ARG GORELEASER_DOWNLOAD_URL=https://github.com/goreleaser/goreleaser/releases/download/v${GORELEASER_VERSION}/${GORELEASER_DOWNLOAD_FILE}

RUN  wget ${GORELEASER_DOWNLOAD_URL}; \
			echo "$GORELEASER_SHA $GORELEASER_DOWNLOAD_FILE" | sha256sum -c - || exit 1; \
			tar -xzf $GORELEASER_DOWNLOAD_FILE -C /usr/bin/ goreleaser; \
			rm $GORELEASER_DOWNLOAD_FILE;

RUN apt-get update && \
    apt-get install -y \
    apt-transport-https \
    ca-certificates \
    curl \
    gnupg-agent \
    software-properties-common

RUN curl -fsSL https://download.docker.com/linux/debian/gpg | apt-key add - && \
  add-apt-repository \
   "deb [arch=amd64] https://download.docker.com/linux/debian \
   $(lsb_release -cs) \
   stable"

RUN apt-get update && \
	apt-get install -y docker-ce \
	docker-ce-cli

ENTRYPOINT ["/entrypoint.sh"]
CMD ["-h"]

COPY scripts/entrypoint.sh /entrypoint.sh
RUN chmod +x /entrypoint.sh

FROM golang:1.8.1

ARG APP_VERSION=unkown
ARG BUILD_DATE=unkown
ARG GIT_REPOSITORY=unkown
ARG GIT_BRANCH=unkown
ARG GIT_COMMIT=unkown
ARG MAINTAINER=unkown

LABEL syros.version=$APP_VERSION \
      syros.build=$BUILD_DATE \
      syros.repository=$GIT_REPOSITORY \
      syros.branch=$GIT_BRANCH \
      syros.revision=$GIT_COMMIT \
      syros.maintainer=$MAINTAINER

EXPOSE 8886

COPY /dist/agent /syros/agent

#RUN apk add --no-cache --virtual curl && chmod 777 /syros/agent
RUN apt-get update && apt-get install -y --no-install-recommends \
		ca-certificates \
		curl \
		wget \
	&& rm -rf /var/lib/apt/lists/*

HEALTHCHECK --interval=30s --timeout=15s --retries=3\
  CMD curl -f http://localhost:8886/status || exit 1

WORKDIR /syros
ENTRYPOINT ["/syros/agent"]


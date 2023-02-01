FROM --platform=$BUILDPLATFORM golang:1.18.10-alpine as go-builder

ARG BUILDPLATFORM
ARG TARGETOS
ARG TARGETARCH

RUN apk update && apk upgrade && \
  apk add --no-cache ca-certificates git mercurial

ARG PROJECT_NAME=redis-cluster-operator
ARG REPO_PATH=github.com/ucloud/$PROJECT_NAME
ARG BUILD_PATH=${REPO_PATH}/cmd/manager

# Build version and commit should be passed in when performing docker build
ARG VERSION=0.1.1
ARG GIT_SHA=0000000

WORKDIR /src

COPY go.mod go.sum ./
RUN --mount=type=cache,id=gomod,target=/go/pkg/mod \
  GOOS=$TARGETOS GOARCH=$TARGETARCH go mod download

COPY pkg ./ cmd ./ version ./

RUN --mount=type=cache,id=gomod,target=/go/pkg/mod \
  --mount=type=cache,id=gobuild,target=/root/.cache/go-build \
  GOOS=$TARGETOS GOARCH=$TARGETARCH CGO_ENABLED=0 go build -o ${GOBIN}/${PROJECT_NAME} \
  -ldflags "-X ${REPO_PATH}/version.Version=${VERSION} -X ${REPO_PATH}/version.GitSHA=${GIT_SHA}" \
  $BUILD_PATH

# =============================================================================
FROM alpine:3.9 AS final

ARG PROJECT_NAME=redis-cluster-operator

COPY --from=go-builder ${GOBIN}/${PROJECT_NAME} /usr/local/bin/${PROJECT_NAME}

RUN adduser -D ${PROJECT_NAME}
USER ${PROJECT_NAME}

ENTRYPOINT ["/usr/local/bin/redis-cluster-operator"]

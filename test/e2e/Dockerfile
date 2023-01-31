ARG GOLANG_VERSION=1.18.10
FROM golang:${GOLANG_VERSION}

ENV GOLANG_VERSION=${GOLANG_VERSION}

RUN apt update && apt install -y git

RUN go install github.com/onsi/ginkgo/ginkgo@latest github.com/onsi/gomega/...@latest

ARG PROJECT_NAME=redis-cluster-operator
ARG REPO_PATH=github.com/ucloud/$PROJECT_NAME

RUN mkdir -p /go/src/${REPO_PATH}
COPY . /go/src/${REPO_PATH}
RUN chmod +x /go/src/${REPO_PATH}/hack/e2e.sh

CMD /go/src/github.com/ucloud/redis-cluster-operator/hack/e2e.sh
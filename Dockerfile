FROM golang:1.12-alpine3.9 as BuildEnv

ENV CGO_ENABLED 0
ENV GOOS linux
ENV PROTOC_VERSION_TAG v1.3.1
ENV PATH $PATH:/root/google-cloud-sdk/bin:/root/

# Install dependencies that change infrequently:
RUN apk add --no-cache --update git python python2-dev openssl-dev libffi-dev py-pip docker gcc musl-dev openssh curl libc6-compat protobuf protobuf-dev make bash which && \
    rm -f /var/cache/apk/*

RUN pip install docker-compose

RUN curl -s -f -L -o await https://github.com/betalo-sweden/await/releases/download/v0.4.0/await-linux-amd64 && \
    chmod +x await  && \
    mv await /root/await

RUN go get -u -d golang.org/x/tools/cmd/goimports && \
    go install golang.org/x/tools/cmd/goimports && \
    go get -u -d github.com/golang/mock/gomock && \
    go install github.com/golang/mock/gomock && \
    go get -u -d github.com/golang/mock/mockgen && \
    go install github.com/golang/mock/mockgen

# Get the golangci-linter
RUN curl -sfL https://install.goreleaser.com/github.com/golangci/golangci-lint.sh | sh -s v1.15.0 && \

    # Workaround for https://github.com/golang/protobuf/issues/763#issuecomment-442767135.
    # Then `checkout master` because `go get` below does `pull --ff-only` in same local repo.
    go get -u -d github.com/golang/protobuf/protoc-gen-go && \
    git -C "$(go env GOPATH)"/src/github.com/golang/protobuf checkout $PROTOC_VERSION_TAG && \
    go install github.com/golang/protobuf/protoc-gen-go && \
    git -C "$(go env GOPATH)"/src/github.com/golang/protobuf checkout master && \

    go get -u -d github.com/golang/protobuf/ptypes && \
    git -C "$(go env GOPATH)"/src/github.com/golang/protobuf checkout $PROTOC_VERSION_TAG && \
    go install github.com/golang/protobuf/ptypes && \
    git -C "$(go env GOPATH)"/src/github.com/golang/protobuf checkout master && \

    go get -u google.golang.org/grpc github.com/kevinburke/go-bindata/... && \

    curl -sSL https://sdk.cloud.google.com | bash

# Build ------------------------------------------------------------------------------------------
FROM BuildEnv AS builder

ARG VERSION=snapshot

WORKDIR /app
ADD . /app

RUN go build -o go-app -mod vendor -ldflags "-X main.version=${VERSION}" ./cmd/unprotected/

# ------------ RUN -----------
FROM alpine:3.8
ENTRYPOINT ["/app/go-app"]
WORKDIR /app
RUN apk --update --no-cache add ca-certificates && update-ca-certificates && rm -f /var/cache/apk/*
COPY --from=builder /app/go-app /app/go-app

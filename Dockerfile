FROM golang:1.13.3-alpine3.10 as builder

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

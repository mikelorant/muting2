FROM alpine:3.16 as base
RUN apk add --no-cache \
      tini
RUN addgroup -g 10001 muting && \
    adduser -S -D -H -h / -s /usr/sbin/nologin -u 10001 muting && \
    adduser muting muting

FROM golang:1.19-alpine3.16 AS dependencies
ENV GOOS=linux
ENV CGO_ENABLED=0
WORKDIR /usr/src/app
COPY go.mod go.sum ./
RUN --mount=type=cache,id=gomod,target=/go/pkg/mod \
    --mount=type=cache,id=gobuild,target=/root/.cache/go-build \
    go mod download

FROM dependencies as build
COPY . ./
RUN --mount=type=cache,id=gomod,target=/go/pkg/mod \
    --mount=type=cache,id=gobuild,target=/root/.cache/go-build \
    go build

FROM base as release
COPY --from=build /usr/src/app/muting2 /usr/local/bin/muting
USER muting
ENTRYPOINT ["tini", "--", "muting"]

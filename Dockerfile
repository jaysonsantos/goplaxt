FROM --platform=$BUILDPLATFORM golang:1.19-alpine as builder
ARG TARGETARCH
WORKDIR $GOPATH/src/github.com/xanderstrike/goplaxt/
RUN apk add --no-cache git
COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    go mod download
COPY . .
RUN mkdir /out
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 GOOS=linux GOARCH=${TARGETARCH} go build -o /out/goplaxt-docker

FROM alpine
LABEL maintainer="xanderstrike@gmail.com"
WORKDIR /app
COPY --from=builder /out .
COPY static ./static
VOLUME /app/keystore/
EXPOSE 8000
ENTRYPOINT ["/app/goplaxt-docker"]

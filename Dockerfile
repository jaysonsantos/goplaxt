FROM golang:1.17-alpine as builder
WORKDIR $GOPATH/src/github.com/xanderstrike/goplaxt/
RUN apk add --no-cache git
COPY . .
RUN mkdir /out
RUN mkdir /out/keystore
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /out/goplaxt-docker

FROM alpine
LABEL maintainer="xanderstrike@gmail.com"
WORKDIR /app
COPY static ./static
VOLUME /app/keystore/
EXPOSE 8000
ENTRYPOINT ["/app/goplaxt-docker"]

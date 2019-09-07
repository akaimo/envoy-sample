FROM golang:1.13 as builder

WORKDIR /go/src/github.com/envoy-sample

COPY go.mod .
COPY go.sum .

RUN set -x \
 && go mod download

COPY . .

RUN set -x \
  && go build .

FROM debian:latest

EXPOSE 80

COPY --from=builder /go/src/github.com/envoy-sample/xds /xds

CMD /xds

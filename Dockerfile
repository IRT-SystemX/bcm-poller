
FROM ubuntu:20.04 as go_builder

RUN apt-get update && apt-get install -y wget gcc

ARG GO_VERSION=1.14

RUN wget https://dl.google.com/go/go${GO_VERSION}.linux-amd64.tar.gz \
    && tar -C /usr/local -xzf go${GO_VERSION}.linux-amd64.tar.gz \
    && rm go${GO_VERSION}.linux-amd64.tar.gz

ENV GOROOT=/usr/local/go
ENV GOPATH=/go
ENV PATH=$GOPATH/bin:$GOROOT/bin:$PATH

WORKDIR /go/src

########################################

FROM go_builder as go_build

COPY . /go/src

RUN GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go install cmd/main.go

########################################

FROM alpine:3.11 as install

COPY --from=go_build /go/bin/main /usr/local/bin/poller

EXPOSE 8000

ENTRYPOINT ["poller"]

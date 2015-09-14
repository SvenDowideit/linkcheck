FROM golang:1.4-cross

ENV GOPATH /go
ENV USER root

WORKDIR /go/src/github.com/SvenDowideit/linkcheck

ADD . /go/src/github.com/SvenDowideit/linkcheck
RUN go get -d -v
RUN go build -o linkcheck linkcheck.go


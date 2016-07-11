FROM golang

ENV GOPATH /go
ENV USER root

RUN go get golang.org/x/net/html
RUN apt-get update \
	&& apt-get install -yq zip

WORKDIR /go/src/github.com/SvenDowideit/linkcheck

ADD . /go/src/github.com/SvenDowideit/linkcheck
RUN go get -d -v
RUN go build -o linkcheck linkcheck.go \
	&& GOOS=windows GOARCH=amd64 go build -o linkcheck.exe linkcheck.go \
	&& GOOS=darwin GOARCH=amd64 go build -o linkcheck.app linkcheck.go \
	&& zip linkcheck.zip linkcheck linkcheck.exe linkcheck.app


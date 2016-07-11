FROM golang

# Simplify making releases
RUN apt-get update \
	&& apt-get install -yq zip bzip2
RUN wget -O github-release.bz2 https://github.com/aktau/github-release/releases/download/v0.6.2/linux-amd64-github-release.tar.bz2 \
        && tar jxvf github-release.bz2 \
        && mv bin/linux/amd64/github-release /usr/local/bin/ \
        && rm github-release.bz2


ENV GOPATH /go
ENV USER root

RUN go get golang.org/x/net/html

WORKDIR /go/src/github.com/SvenDowideit/linkcheck

ADD . /go/src/github.com/SvenDowideit/linkcheck
RUN go get -d -v
RUN go build -o linkcheck linkcheck.go \
	&& GOOS=windows GOARCH=amd64 go build -o linkcheck.exe linkcheck.go \
	&& GOOS=darwin GOARCH=amd64 go build -o linkcheck.app linkcheck.go \
	&& zip linkcheck.zip linkcheck linkcheck.exe linkcheck.app


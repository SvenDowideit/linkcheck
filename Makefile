
# Adds build information from git repo
#
# as suggested by tatsushid in
# https://github.com/spf13/hugo/issues/540

COMMIT_HASH=`git rev-parse --short HEAD 2>/dev/null`
BUILD_DATE=`date +%FT%T%z`
LDFLAGS=-ldflags "-X github.com/spf13/hugo/hugolib.CommitHash=${COMMIT_HASH} -X github.com/spf13/hugo/hugolib.BuildDate=${BUILD_DATE}"

build:
	go build -o hugo main.go

docker:
	rm -f linkcheck.gz
	docker build -t linkcheck .
	docker run --name linkcheck-build linkcheck gzip linkcheck
	docker cp linkcheck-build:/go/src/github.com/SvenDowideit/linkcheck/linkcheck.gz .
	docker rm linkcheck-build
	gunzip linkcheck.gz

run:
	./linkcheck http://10.10.10.20:8000

post:
	./linkcheck https://docs.docker.com

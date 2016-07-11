
# Adds build information from git repo
#
# as suggested by tatsushid in
# https://github.com/spf13/hugo/issues/540

COMMIT_HASH=`git rev-parse --short HEAD 2>/dev/null`
BUILD_DATE=`date +%FT%T%z`
LDFLAGS=-ldflags "-X github.com/spf13/hugo/hugolib.CommitHash=${COMMIT_HASH} -X github.com/spf13/hugo/hugolib.BuildDate=${BUILD_DATE}"

AWSTOKENSFILE ?= ../aws.env
-include $(AWSTOKENSFILE)
export GITHUB_USERNAME GITHUB_TOKEN

build:
	go build -o hugo main.go

docker:
	docker build -t linkcheck .
	rm -f linkcheck.gz
	docker rm linkcheck-build || true
	docker run --name linkcheck-build linkcheck ls
	docker cp linkcheck-build:/go/src/github.com/SvenDowideit/linkcheck/linkcheck.zip .
	unzip -o linkcheck.zip

RELEASE_DATE=`date +%FT%T%z`

release: docker
	# TODO: check that we have upstream master, bail if not
	docker run --rm -it -e GITHUB_TOKEN linkcheck \
		github-release release --user docker --repo linkcheck --tag $(RELEASE_DATE)
	docker run --rm -it -e GITHUB_TOKEN linkcheck \
		github-release upload --user docker --repo linkcheck --tag $(RELEASE_DATE) \
			--name linkcheck \
			--file linkcheck
	docker run --rm -it -e GITHUB_TOKEN linkcheck \
		github-release upload --user docker --repo linkcheck --tag $(RELEASE_DATE) \
			--name linkcheck-osx \
			--file linkcheck.app
	docker run --rm -it -e GITHUB_TOKEN linkcheck \
		github-release upload --user docker --repo linkcheck --tag $(RELEASE_DATE) \
			--name linkcheck.exe \
			--file linkcheck.exe

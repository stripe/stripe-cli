all: test

install-deps:
	GO111MODULE=on go get

update-deps:
	GO111MODULE=on go get -u

lint:
# In travis, we need to install golint explicitly. Don't do this in other
# environments
ifeq ($(ENVIRONMENT), travis)
	go get golang.org/x/lint/golint
	git checkout .
endif
	golint -set_exit_status ./...

vet:
	go vet $(go list ./... | grep -v /vendor/)

test: install-deps lint vet
	go test -race -cover -v ./...
	@echo '\o/ yay, we did it!'

# This does not release anything from your local machine but creates a tag
# for our CI to handle it
release:
	git pull origin master

# Makefile's execute each line in its own subshell so variables don't
# persist. Instead, grab the version and run the `tag` command in the same
# subprocess by escaping the newline
	@read -p "Enter new version (of the format vN.N.N): " version; \
	git tag $$version
	git push --tags

.PHONY: test fmt cov tidy run lint blackboxtest modernize modernize-fix

COVFILE = coverage.out
COVHTML = cover.html

test:
	go test ./... -json | go tool tparse -all

fmt:
	go tool gofumpt -l -w .

cov:
	go test -cover ./... -coverprofile=$(COVFILE)
	go tool cover -html=$(COVFILE) -o $(COVHTML)
	rm $(COVFILE)

tidy:
	go mod tidy -v

lint:
	go tool golangci-lint run -v

ci: fmt modernize-fix lint cov blackboxtest

blackboxtest:
	go build
	./_testscripts/test_functionality.sh

# Go Modernize
modernize:
	go run golang.org/x/tools/gopls/internal/analysis/modernize/cmd/modernize@latest -test ./...

modernize-fix:
	go run golang.org/x/tools/gopls/internal/analysis/modernize/cmd/modernize@latest -fix ./...


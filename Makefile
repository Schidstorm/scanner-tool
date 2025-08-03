project = scanner-tool
arch = $(shell /usr/local/go/bin/go env GOARCH)

all: test build remote

test:
	go test -v ./... -coverprofile=coverage.out && \
	go tool cover -html=coverage.out -o coverage.html

build: dependencies tidy build_arch

dependencies:
	sudo apt install -y tesseract-ocr libsane-dev

tidy:
	/usr/local/go/bin/go mod tidy

build_arch: test
	mkdir -p dist && \
	CGO_ENABLED=1 /usr/local/go/bin/go build -a -o dist/$(project)_$(arch) ./cmd/$(project)

remote:
	bash remoteBuild.sh "$(shell cat remotes.txt)"

install:
	go install ./cmd/$(project)/ && \
	sudo cp systemd/$(project).service /etc/systemd/system/ && \
	sudo systemctl daemon-reload && \
	sudo systemctl enable $(project) && \
	sudo systemctl restart $(project)
	
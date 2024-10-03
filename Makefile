
htmxUrl = https://unpkg.com/htmx.org@latest/dist/
bootstrapUrl = https://api.github.com/repos/twbs/bootstrap/releases/latest
scriptsDir = pkg/server/public

build: update tidy installDependencies
	go build -o dist/server cmd/server/main.go

tidy:
	go mod tidy

installDependencies:
	sudo apt install -y tesseract-ocr libsane-dev

update: updateHtmx updateBootstrap

updateHtmx:
	curl -L -s --fail "$(htmxUrl)/htmx.min.js" > $(scriptsDir)/htmx.min.js

updateBootstrap:
	curl --fail -s -L $(bootstrapUrl) | jq -r '.assets[] | select(.name | endswith("dist.zip")) | .browser_download_url' \
	| xargs -L1 curl --fail -s -L > download.zip && \
	unzip -o -q -j download.zip '*/bootstrap.bundle.min.js*' '*/bootstrap.min.css*' -d $(scriptsDir) && \
	rm download.zip
	
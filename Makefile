
htmxUrl = https://unpkg.com/htmx.org@latest/dist/
bootstrapUrl = https://api.github.com/repos/twbs/bootstrap/releases/latest
scriptsDir = pkg/server/public
remoteCompileSsh = .remote-compile-ssh.sh

install: installDependencies
	go install ./cmd/scanner-tool/ && \
	sudo cp systemd/scanner-tool.service /etc/systemd/system/ && \
	sudo systemctl daemon-reload && \
	sudo systemctl enable scanner-tool && \
	sudo systemctl restart scanner-tool

build: installDependencies
	go build -o dist/scanner-tool ./cmd/scanner-tool/ 

remoteBuild:
	bash $(remoteCompileSsh)

tidy:
	go mod tidy

installDependencies:
	sudo apt install -y tesseract-ocr libsane-dev libsane

update: updateHtmx updateBootstrap

updateHtmx:
	curl -L -s --fail "$(htmxUrl)/htmx.min.js" > $(scriptsDir)/htmx.min.js

updateBootstrap:
	curl --fail -s -L $(bootstrapUrl) | jq -r '.assets[] | select(.name | endswith("dist.zip")) | .browser_download_url' \
	| xargs -L1 curl --fail -s -L > download.zip && \
	unzip -o -q -j download.zip '*/bootstrap.bundle.min.js*' '*/bootstrap.min.css*' -d $(scriptsDir) && \
	rm download.zip
	
build:
	npm install && npm run build
	chmod +x bin/clink.js

install: build
	cp bin/clink.js /usr/local/bin/clink
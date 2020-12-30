build:
	mkdir -p dist
	echo '#!/usr/bin/env deno run -A --unstable\n' > dist/cm
	deno bundle  --unstable ./index.ts >> dist/cm
	chmod +x dist/cm

install: build
	cp dist/cm /usr/local/bin/

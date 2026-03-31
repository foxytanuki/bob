go := env_var_or_default("GO", "go")
prefix := env_var_or_default("PREFIX", home_directory() + "/.local")
bindir := env_var_or_default("BINDIR", prefix + "/bin")

build:
	{{go}} build ./...

build-binaries:
	mkdir -p bin
	{{go}} build -o bin/bob ./cmd/bob
	{{go}} build -o bin/bobd ./cmd/bobd

install: build-binaries
	mkdir -p {{bindir}}
	install -m 755 bin/bob {{bindir}}/bob
	install -m 755 bin/bobd {{bindir}}/bobd

fmt:
	{{go}} fmt ./...

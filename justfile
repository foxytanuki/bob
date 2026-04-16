go := env_var_or_default("GO", "go")
prefix := env_var_or_default("PREFIX", home_directory() + "/.local")
bindir := env_var_or_default("BINDIR", prefix + "/bin")
version := env_var_or_default("VERSION", "v0.2.0")
commit := env_var_or_default("COMMIT", "")
date := env_var_or_default("DATE", "")

build:
	{{go}} build -ldflags "-X bob/internal/version.Version={{version}} -X bob/internal/version.Commit={{commit}} -X bob/internal/version.Date={{date}}" ./...

build-binaries:
	mkdir -p bin
	{{go}} build -ldflags "-X bob/internal/version.Version={{version}} -X bob/internal/version.Commit={{commit}} -X bob/internal/version.Date={{date}}" -o bin/bob ./cmd/bob
	{{go}} build -ldflags "-X bob/internal/version.Version={{version}} -X bob/internal/version.Commit={{commit}} -X bob/internal/version.Date={{date}}" -o bin/bobd ./cmd/bobd

version:
	printf 'VERSION=%s\nCOMMIT=%s\nDATE=%s\n' '{{version}}' '{{commit}}' '{{date}}'

install: build-binaries
	mkdir -p {{bindir}}
	install -m 755 bin/bob {{bindir}}/bob
	install -m 755 bin/bobd {{bindir}}/bobd

fmt:
	{{go}} fmt ./...

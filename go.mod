module github.com/aliumujib/qmd-go

go 1.23.0

require (
	github.com/bmatcuk/doublestar/v4 v4.10.0
	github.com/ebitengine/purego v0.9.1
	github.com/go-skynet/go-llama.cpp v0.0.0-20240314183750-6a8041ef6b46
	github.com/mattn/go-sqlite3 v1.14.33
	github.com/modelcontextprotocol/go-sdk v1.2.0
	github.com/spf13/cobra v1.10.2
	gopkg.in/yaml.v3 v3.0.1
)

require (
	github.com/google/jsonschema-go v0.3.0 // indirect
	github.com/inconshreveable/mousetrap v1.1.0 // indirect
	github.com/spf13/pflag v1.0.10 // indirect
	github.com/yosida95/uritemplate/v3 v3.0.2 // indirect
	golang.org/x/oauth2 v0.30.0 // indirect
)

replace github.com/go-skynet/go-llama.cpp => ./.deps/go-llama.cpp

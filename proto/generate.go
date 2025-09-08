package proto

//go:generate protoc -I. --go_out=go --go_opt=paths=source_relative --go-grpc_out=go --go-grpc_opt=paths=source_relative store.proto

//go:generate sh -c "if command -v grpc_ruby_plugin >/dev/null 2>&1; then protoc -I. --ruby_out=ruby/lib --plugin=protoc-gen-grpc=/opt/homebrew/bin/grpc_ruby_plugin --grpc_out=ruby/lib store.proto; else echo 'Ruby gRPC plugin not found, skipping Ruby generation'; fi"

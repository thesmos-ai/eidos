module go.thesmos.sh/eidos/bridge/protogo

go 1.26.2

require (
	go.thesmos.sh/eidos v1.0.0
	go.thesmos.sh/eidos/frontend/protobuf v1.0.0
)

require (
	github.com/bufbuild/protocompile v0.14.1 // indirect
	golang.org/x/sync v0.20.0 // indirect
	google.golang.org/protobuf v1.34.2 // indirect
)

replace (
	go.thesmos.sh/eidos => ../../
	go.thesmos.sh/eidos/frontend/protobuf => ../../frontend/protobuf
)

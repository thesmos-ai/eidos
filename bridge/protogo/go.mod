module go.thesmos.sh/eidos/bridge/protogo

go 1.26.2

require (
	go.thesmos.sh/eidos v0.0.0-00010101000000-000000000000
	go.thesmos.sh/eidos/frontend/protobuf v0.0.0-00010101000000-000000000000
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

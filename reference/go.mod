module go.thesmos.sh/eidos/reference

go 1.26.2

require (
	go.thesmos.sh/eidos v1.0.0
	go.thesmos.sh/eidos/backend/golang v1.0.0
	go.thesmos.sh/eidos/eidostest v1.0.0
)

require (
	go.thesmos.sh/eidos/frontend/golang v1.0.0 // indirect
	golang.org/x/mod v0.36.0 // indirect
	golang.org/x/sync v0.20.0 // indirect
	golang.org/x/tools v0.45.0 // indirect
)

replace (
	go.thesmos.sh/eidos => ../
	go.thesmos.sh/eidos/backend/golang => ../backend/golang
	go.thesmos.sh/eidos/eidostest => ../eidostest
	go.thesmos.sh/eidos/frontend/golang => ../frontend/golang
)

module app

go 1.20

require (
	github.com/hashicorp/yamux v0.1.1
	github.com/lainio/err2 v0.9.1
	github.com/tetratelabs/wazero v1.2.1
)

replace github.com/hashicorp/yamux => ./yamux

module github.com/aretaja/godevman

go 1.16

// For local development
replace (
	github.com/aretaja/godevman => ./
	github.com/aretaja/snmphelper => ../snmphelper
)

require (
	github.com/PraserX/ipconv v1.1.0
	github.com/aretaja/snmphelper v1.1.1
	github.com/davecgh/go-spew v1.1.1
	github.com/kr/pretty v0.3.0
)

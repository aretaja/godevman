module github.com/aretaja/godevman

go 1.17

// For local development
//replace github.com/aretaja/snmphelper => ../snmphelper

require (
	github.com/aretaja/snmphelper v1.1.2
	github.com/davecgh/go-spew v1.1.1
	github.com/kr/pretty v0.3.0
	github.com/patrickmn/go-cache v2.1.0+incompatible
	github.com/praserx/ipconv v1.2.1
)

require (
	github.com/gosnmp/gosnmp v1.34.0 // indirect
	github.com/kr/text v0.2.0 // indirect
	github.com/rogpeppe/go-internal v1.8.1 // indirect
)

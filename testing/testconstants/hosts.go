package testconstants

import (
	"strconv"
)

var (
	Localhost              = "localhost"
	HTTPLocalhost          = "http://localhost"
	HTTPLocalhostWithPort  = portFunction(HTTPLocalhost)
	HTTPSLocalhost         = "https://localhost"
	HTTPSLocalhostWithPort = portFunction(HTTPSLocalhost)

	Github      = "github.com"
	HTTPGithub  = "http://github.com"
	HTTPSGithub = "https://github.com"

	Stackoverflow      = "stackoverflow.com"
	HTTPStackoverflow  = "http://stackoverflow.com"
	HTTPSStackoverflow = "https://stackoverflow.com"

	APIGithub      = "api.github.com"
	HTTPAPIGithub  = "http://api.github.com"
	HTTPSAPIGithub = "https://api.github.com"

	Localhost1              = "localhost1"
	HTTPLocalhost1          = "http://localhost1"
	HTTPLocalhost1WithPort  = portFunction(HTTPLocalhost1)
	HTTPSLocalhost1         = "https://localhost1"
	HTTPSLocalhost1WithPort = portFunction(HTTPSLocalhost1)

	Localhost2              = "localhost2"
	HTTPLocalhost2          = "http://localhost2"
	HTTPLocalhost2WithPort  = portFunction(HTTPLocalhost2)
	HTTPSLocalhost2         = "https://localhost2"
	HTTPSLocalhost2WithPort = portFunction(HTTPSLocalhost2)

	Localhost3              = "localhost3"
	HTTPLocalhost3          = "http://localhost3"
	HTTPLocalhost3WithPort  = portFunction(HTTPLocalhost3)
	HTTPSLocalhost3         = "https://localhost3"
	HTTPSLocalhost3WithPort = portFunction(HTTPSLocalhost3)

	Localhost4              = "localhost4"
	HTTPLocalhost4          = "http://localhost4"
	HTTPLocalhost4WithPort  = portFunction(HTTPLocalhost4)
	HTTPSLocalhost4         = "https://localhost4"
	HTTPSLocalhost4WithPort = portFunction(HTTPSLocalhost4)
)

func portFunction(host string) func(port int) string {
	return func(port int) string {
		return host + ":" + strconv.Itoa(port)
	}
}

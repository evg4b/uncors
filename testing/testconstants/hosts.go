package testconstants

import (
	"strconv"
)

var (
	Localhost      = "localhost"
	HTTPLocalhost  = "http://localhost"
	HTTPSLocalhost = "https://localhost"

	Github      = "github.com"
	HTTPGithub  = "http://github.com"
	HTTPSGithub = "https://github.com"

	Host1      = "host1"
	HTTPHost1  = "http://host1"
	HTTPSHost1 = "https://host1"
)

func HTTPLocalhostWithPort(port int) string {
	return HTTPLocalhost + ":" + strconv.Itoa(port)
}

func HTTPSLocalhostWithPort(port int) string {
	return HTTPSLocalhost + ":" + strconv.Itoa(port)
}

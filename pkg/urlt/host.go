package urlt

type Host struct {
	Hostname string
	Port     string
}

func (h *Host) String() string {
	return h.Hostname + ":" + h.Port
}

func ParseHost(host string) (Host, error) {
	hostWithPort, err := parseHost("", host)
	if err != nil {
		return Host{}, err
	}

	hostName, port := splitHostPort(hostWithPort)

	return Host{Hostname: hostName, Port: port}, nil
}

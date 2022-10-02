package urlglob

type ReplacePatternOption = func(pattern *ReplacePattern)

func UsePort(port string) ReplacePatternOption {
	return func(pattern *ReplacePattern) {
		pattern.port = port
	}
}

func UseScheme(scheme string) ReplacePatternOption {
	return func(pattern *ReplacePattern) {
		pattern.scheme = scheme
	}
}

package urlglob

type urlGlobOption = func(glob *URLGlob)

type replacePatternOption = func(pattern *ReplacePattern)

func UsePort(port string) replacePatternOption {
	return func(pattern *ReplacePattern) {
		pattern.port = port
	}
}

func UseScheme(scheme string) replacePatternOption {
	return func(pattern *ReplacePattern) {
		pattern.scheme = scheme
	}
}

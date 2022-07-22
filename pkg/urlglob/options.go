package urlglob

type urlGloboption = func(glob *URLGlob)

func SaveOriginalPort() urlGloboption {
	return func(glob *URLGlob) {
		glob.savePort = true
	}
}

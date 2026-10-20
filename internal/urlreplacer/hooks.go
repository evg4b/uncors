package urlreplacer

import "fmt"

func schemeHookFactory(targetScheme string) hook {
	forceScheme := fmt.Sprintf("%s://", targetScheme)

	return func(scheme string) string {
		if scheme != "" {
			return forceScheme
		}

		return scheme
	}
}

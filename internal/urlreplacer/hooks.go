package urlreplacer

import "fmt"

func schemeHookFactory(targetScheme string) hook {
	forceScheme := fmt.Sprintf("%s://", targetScheme)

	return func(scheme string) string {
		if len(scheme) > 0 {
			return forceScheme
		}

		return scheme
	}
}

package urlreplacer

import "github.com/evg4b/uncors/internal/sfmt"

func schemeHookFactory(targetScheme string) hook {
	forceScheme := sfmt.Sprintf("%s://", targetScheme)

	return func(scheme string) string {
		if len(scheme) > 0 {
			return forceScheme
		}

		return scheme
	}
}

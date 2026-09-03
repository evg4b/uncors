package version

// NewVersionIsAvailable is the notice shown when a newer release exists. It
// lives here rather than with the terminal helpers so that checking for a new
// version does not drag the rendering libraries into the service.
const NewVersionIsAvailable = `NEW VERSION IS AVAILABLE!
%s is not the latest version, you should upgrade to %s.
See more information at https://github.com/evg4b/uncors/releases
`

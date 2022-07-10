package handler

type Replcaer interface {
	ToTarget(targetUrl string) (string, error)
	ToSource(targetUrl string, host string) (string, error)
}

type RequestHandelerOptions = func(*RequestHandeler)

func WithTarget(target string) RequestHandelerOptions {
	return func(rh *RequestHandeler) {
		rh.target = target
	}
}

func WithUrlReplcaer(replcaer Replcaer) RequestHandelerOptions {
	return func(rh *RequestHandeler) {
		rh.replcaer = replcaer
	}
}

func WithOrigin(origin string) RequestHandelerOptions {
	return func(rh *RequestHandeler) {
		rh.origin = origin
		rh.origin2 = origin
	}
}

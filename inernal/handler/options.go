package handler

type RequestHandelerOptions = func(*RequestHandeler)

func WithTarget(target string) RequestHandelerOptions {
	return func(rh *RequestHandeler) {
		rh.target = target
	}
}

func WithProtocol(protocol string) RequestHandelerOptions {
	return func(rh *RequestHandeler) {
		rh.protocol = protocol
	}
}

func WithOrigin(origin string) RequestHandelerOptions {
	return func(rh *RequestHandeler) {
		rh.origin = origin
		rh.origin2 = origin
	}
}

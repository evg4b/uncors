package processor

type RequestProcessorOption = func(rp *RequestProcessor)

func WithMiddleware(middleware HandlingMiddleware) RequestProcessorOption {
	return func(rp *RequestProcessor) {
		rp.handlerFunc = middleware.Wrap(rp.handlerFunc)
	}
}

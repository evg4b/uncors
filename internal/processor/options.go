package processor

type requestProcessorOption = func(rp *RequestProcessor)

func WithMiddleware(middleware HandlingMiddleware) requestProcessorOption {
	return func(rp *RequestProcessor) {
		rp.handlerFunc = middleware.Wrap(rp.handlerFunc)
	}
}

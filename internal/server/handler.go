package server

import (
	"context"
	"errors"
	"net/http"

	"github.com/cecobask/fibonacci/internal/api"
	"github.com/cecobask/fibonacci/internal/fibonacci"
)

// handler implements api.StrictServerInterface.
type handler struct{}

var _ api.StrictServerInterface = handler{}

func newHandler() handler {
	return handler{}
}

// GetFibonacci returns the first n Fibonacci numbers.
func (handler) GetFibonacci(_ context.Context, req api.GetFibonacciRequestObject) (api.GetFibonacciResponseObject, error) {
	seq, err := fibonacci.Sequence(req.Params.N)
	switch {
	case errors.Is(err, fibonacci.ErrNegative), errors.Is(err, fibonacci.ErrTooLarge):
		return api.GetFibonacci400ApplicationProblemPlusJSONResponse{
			BadRequestApplicationProblemPlusJSONResponse: api.BadRequestApplicationProblemPlusJSONResponse(
				newProblem(http.StatusBadRequest, err.Error()),
			),
		}, nil
	case err != nil:
		return nil, err
	}

	return api.GetFibonacci200JSONResponse(seq), nil
}

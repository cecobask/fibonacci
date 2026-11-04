package server

import (
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/cecobask/fibonacci/internal/api"
)

const problemContentType = "application/problem+json"

// newProblem builds an RFC 9457 problem with the default "about:blank" type,
// whose title is the HTTP status text.
func newProblem(status int, detail string) api.Problem {
	p := api.Problem{
		Type:   "about:blank",
		Title:  http.StatusText(status),
		Status: status,
	}
	if detail != "" {
		p.Detail = &detail
	}
	return p
}

func writeProblem(w http.ResponseWriter, status int, detail string) {
	w.Header().Set("Content-Type", problemContentType)
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(newProblem(status, detail))
}

// handleRequestError handles errors raised while decoding a request, such as
// a missing or malformed query parameter.
func (s *Server) handleRequestError(w http.ResponseWriter, r *http.Request, err error) {
	s.logger.DebugContext(r.Context(), "invalid request",
		slog.String("request_id", requestIDFromContext(r.Context())),
		slog.Any("error", err),
	)
	writeProblem(w, http.StatusBadRequest, requestErrorDetail(err))
}

// handleResponseError handles unexpected errors returned by a handler.
// The error is logged but never exposed to the client.
func (s *Server) handleResponseError(w http.ResponseWriter, r *http.Request, err error) {
	s.logger.ErrorContext(r.Context(), "request failed",
		slog.String("request_id", requestIDFromContext(r.Context())),
		slog.Any("error", err),
	)
	writeProblem(w, http.StatusInternalServerError, "")
}

// requestErrorDetail turns a request decoding error into a client-facing
// message without leaking parser internals.
func requestErrorDetail(err error) string {
	var (
		required      *api.RequiredParamError
		invalidFormat *api.InvalidParamFormatError
		tooMany       *api.TooManyValuesForParamError
	)
	switch {
	case errors.As(err, &required):
		return fmt.Sprintf("query parameter %q is required", required.ParamName)
	case errors.As(err, &invalidFormat):
		return fmt.Sprintf("query parameter %q has an invalid format", invalidFormat.ParamName)
	case errors.As(err, &tooMany):
		return fmt.Sprintf("query parameter %q must be provided once", tooMany.ParamName)
	default:
		return "invalid request"
	}
}

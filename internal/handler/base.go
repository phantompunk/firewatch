package handler

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
)

type envelope map[string]any

type BaseHandler struct {
	Logger *slog.Logger
}

func (h *BaseHandler) logError(r *http.Request, err error) {
	method := r.Method
	uri := r.URL.RequestURI()

	h.Logger.Error(err.Error(), "method", method, "uri", uri)
}

func (h *BaseHandler) errorResponse(w http.ResponseWriter, r *http.Request, status int, message any) {
	env := envelope{"error": message}

	err := h.writeJSON(w, status, env, nil)
	if err != nil {
		h.logError(r, err)
		w.WriteHeader(500)
	}
}

func (h *BaseHandler) serverErrorResponse(w http.ResponseWriter, r *http.Request, err error) {
	h.logError(r, err)

	message := "the server encountered a problem and could not process your request"
	h.errorResponse(w, r, http.StatusInternalServerError, message)
}


func (h *BaseHandler) writeJSON(w http.ResponseWriter, status int, data any, headers http.Header) error {
	for k, v := range headers {
		for _, value := range v {
			w.Header().Add(k, value)
		}
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	encoder := json.NewEncoder(w)

	if err := encoder.Encode(data); err != nil {
		return err
	}

	return nil
}

func (h *BaseHandler) readJSON(w http.ResponseWriter, r *http.Request, dst any) error {
	r.Body = http.MaxBytesReader(w, r.Body, 1_048_576) // 1MB

	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()

	err := dec.Decode(dst)
	if err != nil {
		var syntaxError *json.SyntaxError
		var unmarshalTypeError *json.UnmarshalTypeError
		var invalidUnmarshalError *json.InvalidUnmarshalError
		var maxBytesError *http.MaxBytesError

		switch {
		case errors.As(err, &syntaxError):
			return fmt.Errorf("body contains badly-formed JSON (at character %d)", syntaxError.Offset)
		case errors.Is(err, io.ErrUnexpectedEOF):
			return errors.New("body contains badly-formed JSON")
		case errors.As(err, &unmarshalTypeError):
			if unmarshalTypeError.Field != "" {
				return fmt.Errorf("body contains incorrect JSON type for field %q", unmarshalTypeError.Field)
			}
			return fmt.Errorf("body contains incorrect JSON type (at character %d)", unmarshalTypeError.Offset)
		case errors.Is(err, io.EOF):
			return errors.New("body must not be empty")
		case errors.As(err, &maxBytesError):
			return fmt.Errorf("body must not be larger than %d bytes", maxBytesError.Limit)
		case errors.As(err, &invalidUnmarshalError):
			panic(err)
		default:
			return err
		}
	}

	// Ensure only a single JSON value is present in the body
	err = dec.Decode(&struct{}{})
	if !errors.Is(err, io.EOF) {
		return errors.New("body must only contain a single JSON value")
	}

	return nil
}

// func (app *application) readJSON(w http.ResponseWriter, r *http.Request, dst any) error {
// 	r.Body = http.MaxBytesReader(w, r.Body, 1_048_576)
//
// 	dec := json.NewDecoder(r.Body)
// 	dec.DisallowUnknownFields()
//
// 	err := dec.Decode(dst)
// 	if err != nil {
// 		var syntaxError *json.SyntaxError
// 		var unmarshalTypeError *json.UnmarshalTypeError
// 		var invalidUnmarshalError *json.InvalidUnmarshalError
// 		var maxBytesError *http.MaxBytesError
//
// 		switch {
// 		case errors.As(err, &syntaxError):
// 			return fmt.Errorf("body contains badly-formed JSON (at character %d)", syntaxError.Offset)
// 		case errors.Is(err, io.ErrUnexpectedEOF):
// 			return errors.New("body contains badly-formed JSON")
// 		case errors.As(err, &unmarshalTypeError):
// 			if unmarshalTypeError.Field != "" {
// 				return fmt.Errorf("body contains incorrect JSON type for field %q", unmarshalTypeError.Field)
// 			}
// 			return fmt.Errorf("body contains incorrect JSON type (at character %d)", unmarshalTypeError.Offset)
// 		case errors.Is(err, io.EOF):
// 			return errors.New("body must not be empty")
// 		case errors.As(err, &maxBytesError):
// 			return fmt.Errorf("body must not be larger than %d bytes", maxBytesError.Limit)
// 		case errors.As(err, &invalidUnmarshalError):
// 			panic(err)
// 		default:
// 			return err
// 		}
// 	}
//
// 	err = dec.Decode(&struct{}{})
// 	if !errors.Is(err, io.EOF) {
// 		return errors.New("body must only contain a single JSON value")
// 	}
// 	return nil
// }
//
//

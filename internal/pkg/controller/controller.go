package controller

import (
	"encoding/json"
	"errors"
	"net/http"
	"regexp"
	"strings"

	"github.com/JLugagne/agach-mcp/pkg/apierror"
	"github.com/go-playground/validator/v10"
	"github.com/sirupsen/logrus"
)

var (
	uuidRe = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)
)

// Controller provides standard HTTP response helpers
type Controller struct {
	logger    *logrus.Logger
	validator *validator.Validate
}

// NewController creates a new controller
func NewController(logger *logrus.Logger) *Controller {
	v := validator.New()
	v.RegisterValidation("entity_id", func(fl validator.FieldLevel) bool {
		s := fl.Field().String()
		return uuidRe.MatchString(s)
	})
	v.RegisterValidation("slug", func(fl validator.FieldLevel) bool {
		s := fl.Field().String()
		if len(s) > 100 {
			return false
		}
		for _, c := range s {
			if !((c >= 'a' && c <= 'z') || (c >= '0' && c <= '9') || c == '-' || c == '_') {
				return false
			}
		}
		return len(s) > 0
	})
	return &Controller{
		logger:    logger,
		validator: v,
	}
}

// Response represents a standard API response
type Response struct {
	Status string      `json:"status"`
	Data   interface{} `json:"data,omitempty"`
	Error  *ErrorData  `json:"error,omitempty"`
}

// ErrorData represents error details
type ErrorData struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// SendSuccess sends a successful response with data
func (c *Controller) SendSuccess(w http.ResponseWriter, r *http.Request, data interface{}) {
	body, err := json.Marshal(Response{
		Status: "success",
		Data:   data,
	})
	if err != nil {
		c.logger.WithError(err).WithField("path", r.URL.Path).Error("Failed to marshal success response")
		c.SendError(w, r, err)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if _, werr := w.Write(body); werr != nil {
		c.logger.WithError(werr).WithField("path", r.URL.Path).Error("Failed to write success response")
	}
}

// CodedError is an interface for domain-level errors that carry a Code and Message.
// It allows the controller to extract structured error information from domain errors
// without creating an import dependency on internal packages.
type CodedError interface {
	ErrorCode() string
	ErrorMessage() string
}

// SendFail sends a failure response (client error - bad request, validation, etc.)
func (c *Controller) SendFail(w http.ResponseWriter, r *http.Request, statusCode *int, err error) {
	code := http.StatusBadRequest
	if statusCode != nil && *statusCode >= 400 && *statusCode < 500 {
		code = *statusCode
	}

	var errCode, errMsg string
	var coded CodedError
	if errors.As(err, &coded) {
		errCode = coded.ErrorCode()
		errMsg = coded.ErrorMessage()
	} else {
		errCode = "CLIENT_ERROR"
		errMsg = "bad request"
	}

	c.logger.WithError(err).WithFields(logrus.Fields{
		"code":   errCode,
		"status": code,
		"path":   r.URL.Path,
	}).Warn("Client error")

	errData := &ErrorData{
		Code:    errCode,
		Message: errMsg,
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	if encErr := json.NewEncoder(w).Encode(Response{
		Status: "fail",
		Data:   errData,
		Error:  errData,
	}); encErr != nil {
		c.logger.WithError(encErr).WithField("path", r.URL.Path).Error("Failed to encode fail response")
	}
}

// SendError sends an error response (server error - unexpected errors)
func (c *Controller) SendError(w http.ResponseWriter, r *http.Request, err error) {
	c.logger.WithError(err).WithField("path", r.URL.Path).Error("Server error")

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusInternalServerError)
	if encErr := json.NewEncoder(w).Encode(Response{
		Status: "error",
		Error: &ErrorData{
			Code:    "INTERNAL_ERROR",
			Message: "An internal error occurred",
		},
	}); encErr != nil {
		c.logger.WithError(encErr).WithField("path", r.URL.Path).Error("Failed to encode error response")
	}
}

// Validate validates a struct using validator tags
func (c *Controller) Validate(data interface{}) error {
	return c.validator.Struct(data)
}

// DecodeAndValidate decodes JSON and validates it.
// Returns an error if Content-Type is not application/json.
// Returns an http.MaxBytesError if the request body exceeds the configured limit.
func (c *Controller) DecodeAndValidate(r *http.Request, data interface{}, validationErr *apierror.Error) error {
	ct := r.Header.Get("Content-Type")
	if !strings.HasPrefix(ct, "application/json") {
		return &apierror.Error{Code: "INVALID_CONTENT_TYPE", Message: "Content-Type must be application/json"}
	}

	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(data); err != nil {
		if strings.Contains(err.Error(), "json: unknown field") {
			return &apierror.Error{Code: "INVALID_REQUEST_BODY", Message: "invalid request body"}
		}
		return err
	}

	if dec.More() {
		return &apierror.Error{Code: "INVALID_REQUEST_BODY", Message: "invalid request body: trailing data"}
	}

	if err := c.Validate(data); err != nil {
		if validationErr != nil {
			return &apierror.Error{
				Code:    validationErr.Code,
				Message: validationErr.Message,
			}
		}
		return err
	}

	return nil
}

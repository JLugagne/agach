package controller

import (
	"encoding/json"
	"net/http"
	"regexp"

	"github.com/JLugagne/agach-mcp/internal/kanban/domain"
	"github.com/go-playground/validator/v10"
	"github.com/sirupsen/logrus"
)

var (
	uuidRe    = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)
	shortIDRe = regexp.MustCompile(`^[0-9a-fA-F]{8}$`)
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
		return uuidRe.MatchString(s) || shortIDRe.MatchString(s)
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
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(Response{
		Status: "success",
		Data:   data,
	})
}

// SendFail sends a failure response (client error - bad request, validation, etc.)
func (c *Controller) SendFail(w http.ResponseWriter, r *http.Request, statusCode *int, err error) {
	code := http.StatusBadRequest
	if statusCode != nil {
		code = *statusCode
	}

	var errCode, errMsg string
	if domainErr, ok := err.(*domain.Error); ok {
		errCode = domainErr.Code
		errMsg = domainErr.Message
	} else {
		errCode = "CLIENT_ERROR"
		errMsg = err.Error()
	}

	c.logger.WithError(err).WithFields(logrus.Fields{
		"code":   errCode,
		"status": code,
		"path":   r.URL.Path,
	}).Warn("Client error")

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(Response{
		Status: "fail",
		Error: &ErrorData{
			Code:    errCode,
			Message: errMsg,
		},
	})
}

// SendError sends an error response (server error - unexpected errors)
func (c *Controller) SendError(w http.ResponseWriter, r *http.Request, err error) {
	c.logger.WithError(err).WithField("path", r.URL.Path).Error("Server error")

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusInternalServerError)
	json.NewEncoder(w).Encode(Response{
		Status: "error",
		Error: &ErrorData{
			Code:    "INTERNAL_ERROR",
			Message: "An internal error occurred",
		},
	})
}

// Validate validates a struct using validator tags
func (c *Controller) Validate(data interface{}) error {
	return c.validator.Struct(data)
}

// DecodeAndValidate decodes JSON and validates it
func (c *Controller) DecodeAndValidate(r *http.Request, data interface{}, validationErr *domain.Error) error {
	if err := json.NewDecoder(r.Body).Decode(data); err != nil {
		return err
	}

	if err := c.Validate(data); err != nil {
		if validationErr != nil {
			validationErr.Err = err
			return validationErr
		}
		return err
	}

	return nil
}

package handlers

import (
	"encoding/json"
	"errors"
	"net/http"
	"sort"
	"strings"

	userpb "services/user/proto"

	"github.com/go-playground/validator/v10"
	"github.com/gorilla/schema"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

type errorResponse struct {
	Error   string       `json:"error"`
	Message string       `json:"message"`
	Fields  []fieldError `json:"fields,omitempty"`
}

type fieldError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

func authenticatedUserIDFromRequest(r *http.Request) (int64, error) {
	authHeader := strings.TrimSpace(r.Header.Get(headerAuth))
	if authHeader == "" {
		return 0, errors.New(errMissingAuthHeader)
	}

	parts := strings.SplitN(authHeader, " ", 2)
	if len(parts) != 2 || !strings.EqualFold(parts[0], authSchemeBearer) || strings.TrimSpace(parts[1]) == "" {
		return 0, errors.New(errInvalidAuthHeader)
	}

	idToken := parts[1]

	if firebaseAuthClient == nil {
		return 0, errors.New("firebase auth client not initialised")
	}

	ctx := r.Context()

	token, err := firebaseAuthClient.VerifyIDToken(ctx, idToken)
	if err != nil {
		println("Firebase token verification failed:", err.Error())
		return 0, errors.New(errInvalidToken)
	}

	firebaseUID := token.UID

	email, _ := token.Claims["email"].(string)
	name, _ := token.Claims["name"].(string)

	resp, err := userClient.GetOrCreateByFirebaseUID(ctx, &userpb.GetOrCreateByFirebaseUIDRequest{
		FirebaseUid: firebaseUID,
		Email:       email,
		Name:        name,
	})
	if err != nil {
		println("Failed to resolve user id via user-service:", err.Error())
		return 0, errors.New("failed to resolve user id")
	}

	return resp.UserId, nil
}

func writeJSON(w http.ResponseWriter, status int, payload interface{}) {
	w.Header().Set(headerContentType, contentTypeJSON)
	w.WriteHeader(status)

	if msg, ok := payload.(proto.Message); ok {
		marshaler := protojson.MarshalOptions{
			EmitUnpopulated: true,
			UseProtoNames:   true,
		}
		b, err := marshaler.Marshal(msg)
		if err == nil {
			_, _ = w.Write(b)
			return
		}
	}

	_ = json.NewEncoder(w).Encode(payload)
}

func writeServiceError(w http.ResponseWriter, err error) {
	message := err.Error()
	if st, ok := status.FromError(err); ok {
		message = st.Message()
	}
	writeError(w, httpStatusFromServiceError(err), message, nil)
}

func writeRequestError(w http.ResponseWriter, message string, err error) {
	writeError(w, http.StatusBadRequest, message, errorFieldsFromError(err))
}

func writeError(w http.ResponseWriter, status int, message string, fields []fieldError) {
	text := strings.ToLower(http.StatusText(status))
	text = strings.ReplaceAll(text, " ", "_")
	writeJSON(w, status, errorResponse{
		Error:   text,
		Message: message,
		Fields:  fields,
	})
}

func errorFieldsFromError(err error) []fieldError {
	if err == nil {
		return nil
	}

	var validationErrs validator.ValidationErrors
	if errors.As(err, &validationErrs) {
		fields := make([]fieldError, 0, len(validationErrs))
		for _, validationErr := range validationErrs {
			fields = append(fields, fieldError{
				Field:   validationErr.Field(),
				Message: validationErrorMessage(validationErr),
			})
		}
		sort.SliceStable(fields, func(i, j int) bool {
			return fields[i].Field < fields[j].Field
		})
		return fields
	}

	var schemaErrs schema.MultiError
	if errors.As(err, &schemaErrs) {
		keys := make([]string, 0, len(schemaErrs))
		for key := range schemaErrs {
			keys = append(keys, key)
		}
		sort.Strings(keys)

		fields := make([]fieldError, 0, len(keys))
		for _, key := range keys {
			fields = append(fields, schemaFieldError(key, schemaErrs[key]))
		}
		return fields
	}

	return nil
}

func validationErrorMessage(err validator.FieldError) string {
	switch err.Tag() {
	case "gt":
		return "must be greater than " + err.Param()
	case "gte":
		return "must be greater than or equal to " + err.Param()
	case "lte":
		return "must be less than or equal to " + err.Param()
	case "min":
		return "must contain at least " + err.Param() + " item(s)"
	case "oneof":
		return "must be one of: " + err.Param()
	case "notblank":
		return "is required"
	case "required":
		return "is required"
	default:
		return "is invalid"
	}
}

func schemaFieldError(key string, err error) fieldError {
	field := key

	var conversionErr schema.ConversionError
	if errors.As(err, &conversionErr) {
		if conversionErr.Key != "" {
			field = conversionErr.Key
		}
		message := "must be a valid value"
		if conversionErr.Type != nil {
			message = "must be a valid " + conversionErr.Type.String()
		}
		return fieldError{
			Field:   field,
			Message: message,
		}
	}

	return fieldError{
		Field:   field,
		Message: "is invalid",
	}
}

func httpStatusFromServiceError(err error) int {
	switch status.Code(err) {
	case codes.InvalidArgument, codes.FailedPrecondition, codes.OutOfRange:
		return http.StatusBadRequest
	case codes.Unauthenticated:
		return http.StatusUnauthorized
	case codes.PermissionDenied:
		return http.StatusForbidden
	case codes.NotFound:
		return http.StatusNotFound
	case codes.AlreadyExists, codes.Aborted:
		return http.StatusConflict
	case codes.Unavailable:
		return http.StatusBadGateway
	default:
		return http.StatusInternalServerError
	}
}

package handlers

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	userpb "services/user/proto"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

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

func decodeJSONBody(r *http.Request, dst interface{}) error {
	contentType := strings.ToLower(strings.TrimSpace(r.Header.Get(headerContentType)))
	if contentType == "" || !strings.HasPrefix(contentType, contentTypeJSON) {
		return errors.New("Content-Type must be application/json")
	}

	return json.NewDecoder(r.Body).Decode(dst)
}

func writeServiceError(w http.ResponseWriter, err error) {
	http.Error(w, err.Error(), httpStatusFromServiceError(err))
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

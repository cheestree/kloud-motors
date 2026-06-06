package handlers

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"

	userpb "services/user/proto"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
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
		b, err := marshalProtoJSONWithNumericInt64(msg)
		if err == nil {
			_, _ = w.Write(b)
			return
		}
	}

	_ = json.NewEncoder(w).Encode(payload)
}

func marshalProtoJSONWithNumericInt64(msg proto.Message) ([]byte, error) {
	marshaler := protojson.MarshalOptions{
		EmitUnpopulated: true,
		UseProtoNames:   true,
	}
	b, err := marshaler.Marshal(msg)
	if err != nil {
		return nil, err
	}

	decoder := json.NewDecoder(bytes.NewReader(b))
	decoder.UseNumber()

	var value interface{}
	if err := decoder.Decode(&value); err != nil {
		return nil, err
	}

	convertInt64Strings(value, msg.ProtoReflect().Descriptor())
	return json.Marshal(value)
}

func convertInt64Strings(value interface{}, descriptor protoreflect.MessageDescriptor) {
	obj, ok := value.(map[string]interface{})
	if !ok {
		return
	}

	fields := descriptor.Fields()
	for i := 0; i < fields.Len(); i++ {
		field := fields.Get(i)
		key := string(field.Name())
		fieldValue, exists := obj[key]
		if !exists {
			continue
		}
		obj[key] = convertFieldInt64Strings(fieldValue, field)
	}
}

func convertFieldInt64Strings(value interface{}, field protoreflect.FieldDescriptor) interface{} {
	if field.IsList() {
		items, ok := value.([]interface{})
		if !ok {
			return value
		}
		for i, item := range items {
			items[i] = convertSingularFieldInt64String(item, field)
		}
		return items
	}

	if field.IsMap() {
		entries, ok := value.(map[string]interface{})
		if !ok {
			return value
		}
		valueField := field.MapValue()
		for key, entry := range entries {
			entries[key] = convertSingularFieldInt64String(entry, valueField)
		}
		return entries
	}

	return convertSingularFieldInt64String(value, field)
}

func convertSingularFieldInt64String(value interface{}, field protoreflect.FieldDescriptor) interface{} {
	switch field.Kind() {
	case protoreflect.Int64Kind, protoreflect.Sint64Kind, protoreflect.Sfixed64Kind:
		return numericJSONNumber(value, false)
	case protoreflect.Uint64Kind, protoreflect.Fixed64Kind:
		return numericJSONNumber(value, true)
	case protoreflect.MessageKind, protoreflect.GroupKind:
		if obj, ok := value.(map[string]interface{}); ok {
			convertInt64Strings(obj, field.Message())
		}
	}
	return value
}

func numericJSONNumber(value interface{}, unsigned bool) interface{} {
	raw, ok := value.(string)
	if !ok {
		return value
	}

	if unsigned {
		if _, err := strconv.ParseUint(raw, 10, 64); err != nil {
			return value
		}
	} else if _, err := strconv.ParseInt(raw, 10, 64); err != nil {
		return value
	}

	return json.Number(raw)
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

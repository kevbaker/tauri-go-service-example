package bridge

import (
	"context"
	"crypto/subtle"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"mime"
	"net/http"
	"strings"
	"time"
)

const (
	MaxRequestBytes  int64 = 64 << 10
	MaxResponseBytes       = 512 << 10
)

type HTTPHandler struct {
	dispatcher     *Dispatcher
	token          string
	allowedOrigins map[string]struct{}
	requestTimeout time.Duration
	shutdown       func()
	logger         *slog.Logger
}

func NewHTTPHandler(dispatcher *Dispatcher, token string, allowedOrigins []string, requestTimeout time.Duration, shutdown func(), logger *slog.Logger) http.Handler {
	origins := make(map[string]struct{}, len(allowedOrigins))
	for _, origin := range allowedOrigins {
		origins[origin] = struct{}{}
	}
	if logger == nil {
		logger = slog.New(slog.NewTextHandler(io.Discard, nil))
	}
	handler := &HTTPHandler{dispatcher: dispatcher, token: token, allowedOrigins: origins, requestTimeout: requestTimeout, shutdown: shutdown, logger: logger}
	mux := http.NewServeMux()
	mux.HandleFunc("POST /v1/invoke", handler.invoke)
	mux.HandleFunc("GET /health", handler.health)
	mux.HandleFunc("POST /v1/shutdown", handler.stop)
	return handler.cors(mux)
}

func (h *HTTPHandler) invoke(response http.ResponseWriter, request *http.Request) {
	if !h.authorize(response, request) || !requireJSON(response, request) {
		return
	}
	if request.ContentLength > MaxRequestBytes {
		writeHTTPError(response, http.StatusRequestEntityTooLarge, "REQUEST_TOO_LARGE", "Request body is too large")
		return
	}
	request.Body = http.MaxBytesReader(response, request.Body, MaxRequestBytes)
	var envelope Request
	decoder := json.NewDecoder(request.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&envelope); err != nil {
		var tooLarge *http.MaxBytesError
		if errors.As(err, &tooLarge) {
			writeHTTPError(response, http.StatusRequestEntityTooLarge, "REQUEST_TOO_LARGE", "Request body is too large")
			return
		}
		writeHTTPError(response, http.StatusBadRequest, "MALFORMED_REQUEST", "Request body is not a valid envelope")
		return
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		var tooLarge *http.MaxBytesError
		if errors.As(err, &tooLarge) {
			writeHTTPError(response, http.StatusRequestEntityTooLarge, "REQUEST_TOO_LARGE", "Request body is too large")
			return
		}
		writeHTTPError(response, http.StatusBadRequest, "MALFORMED_REQUEST", "Request body must contain one JSON value")
		return
	}
	ctx, cancel := requestContext(request, h.requestTimeout)
	defer cancel()
	result := h.dispatcher.Invoke(ctx, envelope)
	encoded, err := json.Marshal(result)
	if err != nil || len(encoded) > MaxResponseBytes {
		h.logger.ErrorContext(ctx, "bridge response rejected", "component", "go-service", "event", "bridge.response.rejected", "requestId", envelope.RequestID)
		writeHTTPError(response, http.StatusInternalServerError, "RESPONSE_INVALID", "Service response could not be encoded")
		return
	}
	response.Header().Set("Content-Type", "application/json")
	response.WriteHeader(http.StatusOK)
	_, _ = response.Write(encoded)
}

func (h *HTTPHandler) health(response http.ResponseWriter, request *http.Request) {
	if !h.authorize(response, request) {
		return
	}
	response.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(response).Encode(map[string]any{"status": "ready", "protocolVersion": ProtocolVersion})
}

func (h *HTTPHandler) stop(response http.ResponseWriter, request *http.Request) {
	if !h.authorize(response, request) {
		return
	}
	response.WriteHeader(http.StatusAccepted)
	if h.shutdown != nil {
		h.shutdown()
	}
}

func (h *HTTPHandler) authorize(response http.ResponseWriter, request *http.Request) bool {
	const prefix = "Bearer "
	header := request.Header.Get("Authorization")
	provided := ""
	if strings.HasPrefix(header, prefix) {
		provided = strings.TrimPrefix(header, prefix)
	}
	if len(provided) != len(h.token) || subtle.ConstantTimeCompare([]byte(provided), []byte(h.token)) != 1 {
		response.Header().Set("WWW-Authenticate", "Bearer")
		writeHTTPError(response, http.StatusUnauthorized, "UNAUTHORIZED", "A valid access token is required")
		return false
	}
	return true
}

func (h *HTTPHandler) cors(next http.Handler) http.Handler {
	return http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		origin := request.Header.Get("Origin")
		if origin != "" {
			if _, allowed := h.allowedOrigins[origin]; !allowed {
				writeHTTPError(response, http.StatusForbidden, "ORIGIN_FORBIDDEN", "Origin is not allowed")
				return
			}
			response.Header().Set("Access-Control-Allow-Origin", origin)
			response.Header().Set("Vary", "Origin")
			response.Header().Set("Access-Control-Allow-Headers", "Authorization, Content-Type")
			response.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		}
		if request.Method == http.MethodOptions {
			response.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(response, request)
	})
}

func requireJSON(response http.ResponseWriter, request *http.Request) bool {
	mediaType, _, err := mime.ParseMediaType(request.Header.Get("Content-Type"))
	if err != nil || mediaType != "application/json" {
		writeHTTPError(response, http.StatusUnsupportedMediaType, "UNSUPPORTED_MEDIA_TYPE", "Content-Type must be application/json")
		return false
	}
	return true
}

func writeHTTPError(response http.ResponseWriter, status int, code, message string) {
	response.Header().Set("Content-Type", "application/json")
	response.WriteHeader(status)
	_ = json.NewEncoder(response).Encode(map[string]any{"error": map[string]string{"code": code, "message": message}})
}

func requestContext(request *http.Request, timeout time.Duration) (context.Context, context.CancelFunc) {
	return context.WithTimeout(request.Context(), timeout)
}

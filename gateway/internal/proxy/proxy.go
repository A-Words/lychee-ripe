package proxy

import (
	"crypto/tls"
	"io"
	"log/slog"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
	"time"

	"github.com/lychee-ripe/gateway/internal/config"
)

// New creates a reverse proxy that forwards requests to the upstream FastAPI
// service. It handles both regular HTTP and WebSocket upgrade requests.
func New(cfg config.UpstreamConfig, logger *slog.Logger) (http.Handler, error) {
	target, err := url.Parse(cfg.BaseURL)
	if err != nil {
		return nil, err
	}

	transport := &http.Transport{
		DialContext: (&net.Dialer{
			Timeout:   time.Duration(cfg.TimeoutS) * time.Second,
			KeepAlive: 30 * time.Second,
		}).DialContext,
		MaxIdleConns:          100,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ResponseHeaderTimeout: time.Duration(cfg.TimeoutS) * time.Second,
	}

	rp := &httputil.ReverseProxy{
		Director: func(req *http.Request) {
			req.URL.Scheme = target.Scheme
			req.URL.Host = target.Host
			req.Host = target.Host

			// Preserve original path as-is; gateway mounts on /v1 prefix.
			if _, ok := req.Header["User-Agent"]; !ok {
				req.Header.Set("User-Agent", "lychee-gateway/1.0")
			}
		},
		Transport: transport,
		ErrorHandler: func(w http.ResponseWriter, r *http.Request, err error) {
			logger.Error("proxy error",
				"path", r.URL.Path,
				"error", err,
			)
			http.Error(w, `{"error":"upstream unavailable"}`, http.StatusBadGateway)
		},
	}

	// Wrap to handle WebSocket upgrades separately.
	return &handler{
		proxy:   rp,
		target:  target,
		timeout: time.Duration(cfg.TimeoutS) * time.Second,
		logger:  logger,
	}, nil
}

type handler struct {
	proxy   *httputil.ReverseProxy
	target  *url.URL
	timeout time.Duration
	logger  *slog.Logger
}

func (h *handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if isWebSocket(r) {
		h.handleWebSocket(w, r)
		return
	}
	h.proxy.ServeHTTP(w, r)
}

func isWebSocket(r *http.Request) bool {
	return strings.EqualFold(r.Header.Get("Connection"), "Upgrade") &&
		strings.EqualFold(r.Header.Get("Upgrade"), "websocket")
}

// handleWebSocket tunnels a WebSocket connection to the upstream by hijacking
// the client connection and dialing the upstream, then copying bytes bidirectionally.
func (h *handler) handleWebSocket(w http.ResponseWriter, r *http.Request) {

	// Dial upstream, using TLS when the upstream scheme is HTTPS.
	rawConn, err := net.DialTimeout("tcp", h.target.Host, h.timeout)
	if err != nil {
		h.logger.Error("ws: dial upstream failed", "error", err)
		http.Error(w, `{"error":"upstream unavailable"}`, http.StatusBadGateway)
		return
	}
	defer rawConn.Close()

	var upConn net.Conn = rawConn
	if h.target.Scheme == "https" {
		host := h.target.Hostname()
		tlsConn := tls.Client(rawConn, &tls.Config{ServerName: host})
		if err := tlsConn.HandshakeContext(r.Context()); err != nil {
			h.logger.Error("ws: TLS handshake failed", "error", err)
			http.Error(w, `{"error":"upstream TLS handshake failed"}`, http.StatusBadGateway)
			return
		}
		upConn = tlsConn
	}

	r.URL.Host = h.target.Host
	r.URL.Scheme = h.target.Scheme
	r.Host = h.target.Host

	if err := r.Write(upConn); err != nil {
		h.logger.Error("ws: write request to upstream failed", "uri", r.URL.RequestURI(), "error", err)
		http.Error(w, `{"error":"upstream unavailable"}`, http.StatusBadGateway)
		return
	}

	// Hijack client connection.
	hj, ok := w.(http.Hijacker)
	if !ok {
		h.logger.Error("ws: hijack not supported")
		http.Error(w, `{"error":"websocket not supported"}`, http.StatusInternalServerError)
		return
	}
	clientConn, _, err := hj.Hijack()
	if err != nil {
		h.logger.Error("ws: hijack failed", "error", err)
		return
	}
	defer clientConn.Close()

	// Bidirectional copy.
	done := make(chan struct{}, 2)
	go func() {
		_, _ = io.Copy(upConn, clientConn)
		done <- struct{}{}
	}()
	go func() {
		_, _ = io.Copy(clientConn, upConn)
		done <- struct{}{}
	}()
	<-done
}

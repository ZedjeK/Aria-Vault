package tlsproxy

import (
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"golang.org/x/net/websocket"
)

func TestHandlerHeaders(t *testing.T) {
	var got *http.Request
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got = r
		io.WriteString(w, "ok")
	}))
	defer backend.Close()
	target, _ := url.Parse(backend.URL)

	front := httptest.NewServer(newHandler(target))
	defer front.Close()

	req, _ := http.NewRequest("GET", front.URL+"/api/sync?x=1", nil)
	req.Host = "vault.exemple.duckdns.org"
	req.Header.Set("X-Real-IP", "6.6.6.6")       // tentative d'usurpation
	req.Header.Set("X-Forwarded-For", "7.7.7.7") // idem
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()

	if got.Host != "vault.exemple.duckdns.org" {
		t.Errorf("Host = %q, l'hôte demandé doit être conservé", got.Host)
	}
	if ip := got.Header.Get("X-Real-IP"); ip != "127.0.0.1" {
		t.Errorf("X-Real-IP = %q, attendu l'IP réelle 127.0.0.1", ip)
	}
	if xff := got.Header.Get("X-Forwarded-For"); xff != "127.0.0.1" {
		t.Errorf("X-Forwarded-For = %q, attendu 127.0.0.1 seul", xff)
	}
	if got.URL.RequestURI() != "/api/sync?x=1" {
		t.Errorf("chemin = %q", got.URL.RequestURI())
	}
}

// Les notifications temps réel de Bitwarden passent par WebSocket.
func TestHandlerWebSocket(t *testing.T) {
	backend := httptest.NewServer(websocket.Handler(func(c *websocket.Conn) {
		io.Copy(c, c) // écho
	}))
	defer backend.Close()
	target, _ := url.Parse(backend.URL)
	front := httptest.NewServer(newHandler(target))
	defer front.Close()

	ws, err := websocket.Dial(strings.Replace(front.URL, "http", "ws", 1)+"/notifications/hub", "", front.URL)
	if err != nil {
		t.Fatal(err)
	}
	defer ws.Close()
	if _, err := ws.Write([]byte("bonjour")); err != nil {
		t.Fatal(err)
	}
	buf := make([]byte, 16)
	n, err := ws.Read(buf)
	if err != nil {
		t.Fatal(err)
	}
	if string(buf[:n]) != "bonjour" {
		t.Errorf("écho WebSocket = %q", buf[:n])
	}
}

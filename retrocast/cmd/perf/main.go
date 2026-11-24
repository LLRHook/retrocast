// Command perf measures the deployed HTTP and gateway paths on a disposable stack.
package main

import (
	"bytes"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"math"
	mathrand "math/rand"
	"net/http"
	"os"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

type apiClient struct {
	base, token, refreshToken string
	http                      *http.Client
}

func (c apiClient) request(method, path string, body any, setup bool) (json.RawMessage, error) {
	for {
		b, err := json.Marshal(body)
		if err != nil {
			return nil, err
		}
		req, err := http.NewRequest(method, c.base+"/api/v1"+path, bytes.NewReader(b))
		if err != nil {
			return nil, err
		}
		req.Header.Set("Content-Type", "application/json")
		if c.token != "" {
			req.Header.Set("Authorization", "Bearer "+c.token)
		}
		res, err := c.http.Do(req)
		if err != nil {
			return nil, err
		}
		raw, err := io.ReadAll(res.Body)
		_ = res.Body.Close()
		if err != nil {
			return nil, err
		}
		if setup && res.StatusCode == 429 {
			s, _ := strconv.Atoi(res.Header.Get("Retry-After"))
			if s < 1 {
				s = 1
			}
			time.Sleep(time.Duration(s) * time.Second)
			continue
		}
		if res.StatusCode >= 300 {
			return nil, fmt.Errorf("%s %s: HTTP %d", method, path, res.StatusCode)
		}
		var envelope struct {
			Data json.RawMessage `json:"data"`
		}
		err = json.Unmarshal(raw, &envelope)
		return envelope.Data, err
	}
}
func (c *apiClient) refresh() {
	auth := decode[struct {
		Token   string `json:"access_token"`
		Refresh string `json:"refresh_token"`
	}](c.request("POST", "/auth/refresh", map[string]string{"refresh_token": c.refreshToken}, true))
	c.token, c.refreshToken = auth.Token, auth.Refresh
}

func decode[T any](raw json.RawMessage, err error) T {
	if err != nil {
		panic(err)
	}
	var v T
	if err = json.Unmarshal(raw, &v); err != nil {
		panic(err)
	}
	return v
}

type entity struct {
	ID   string `json:"id"`
	Code string `json:"code"`
	Type int    `json:"type"`
}
type delivery struct {
	sender int
	sent   time.Time
	seen   map[int]bool
}
type measurement struct {
	mu                                  sync.Mutex
	pending                             map[string]*delivery
	latencies                           []float64
	duplicates, unexpected, disconnects int
	closed                              bool
}

func (m *measurement) receive(client int, content string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	d := m.pending[content]
	if d == nil {
		if strings.HasPrefix(content, "load-") {
			m.unexpected++
		}
		return
	}
	if client == d.sender {
		return
	}
	if d.seen[client] {
		m.duplicates++
		return
	}
	d.seen[client] = true
	m.latencies = append(m.latencies, float64(time.Since(d.sent).Microseconds())/1000)
}
func quantile(values []float64) any {
	if len(values) == 0 {
		return nil
	}
	return p95(values)
}
func p95(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}
	sort.Float64s(values)
	return values[int(math.Ceil(.95*float64(len(values))))-1]
}
func connect(c apiClient, index int, m *measurement) (*websocket.Conn, error) {
	url := "ws" + strings.TrimPrefix(c.base, "http") + "/gateway"
	ws, _, err := websocket.DefaultDialer.Dial(url, nil)
	if err != nil {
		return nil, err
	}
	_ = ws.SetReadDeadline(time.Now().Add(15 * time.Second))
	var hello struct {
		Op int `json:"op"`
	}
	if err = ws.ReadJSON(&hello); err != nil {
		_ = ws.Close()
		return nil, err
	}
	if hello.Op != 10 {
		_ = ws.Close()
		return nil, fmt.Errorf("missing HELLO")
	}
	if err = ws.WriteJSON(map[string]any{"op": 2, "d": map[string]string{"token": c.token}}); err != nil {
		_ = ws.Close()
		return nil, err
	}
	for {
		var e struct {
			Event string `json:"t"`
		}
		if err = ws.ReadJSON(&e); err != nil {
			_ = ws.Close()
			return nil, err
		}
		if e.Event == "READY" {
			break
		}
	}
	_ = ws.SetReadDeadline(time.Time{})
	go func() {
		for {
			var e struct {
				Op    int    `json:"op"`
				Event string `json:"t"`
				Data  struct {
					Content string `json:"content"`
				} `json:"d"`
			}
			if err := ws.ReadJSON(&e); err != nil {
				m.mu.Lock()
				if !m.closed {
					m.disconnects++
				}
				m.mu.Unlock()
				return
			}
			if e.Op == 1 {
				_ = ws.WriteJSON(map[string]any{"op": 1, "d": nil})
			}
			if e.Event == "MESSAGE_CREATE" {
				m.receive(index, e.Data.Content)
			}
		}
	}()
	return ws, nil
}
func main() {
	base := flag.String("base", "http://localhost:8080", "Deployed API or Caddy HTTP URL; disposable open-registration stack only")
	duration := flag.Duration("duration", 10*time.Minute, "Load duration (acceptance requires 10m)")
	flag.Parse()
	if *duration <= 0 {
		panic("duration must be positive")
	}
	c := apiClient{base: strings.TrimRight(*base, "/"), http: &http.Client{Timeout: 10 * time.Second}}
	nonce := make([]byte, 8)
	if _, err := rand.Read(nonce); err != nil {
		panic(err)
	}
	run := hex.EncodeToString(nonce)
	clients := make([]apiClient, 20)
	for i := range clients {
		auth := decode[struct {
			Token   string `json:"access_token"`
			Refresh string `json:"refresh_token"`
		}](c.request("POST", "/auth/register", map[string]string{"username": fmt.Sprintf("perf_%s_%d", run, i), "password": run + "Aa9!"}, true))
		clients[i] = c
		clients[i].token = auth.Token
		clients[i].refreshToken = auth.Refresh
	}
	fmt.Println("setup: 20 users registered (429 Retry-After honored only during setup)")
	guild := decode[entity](clients[0].request("POST", "/guilds", map[string]string{"name": "Perf " + run}, true))
	invite := decode[entity](clients[0].request("POST", "/guilds/"+guild.ID+"/invites", map[string]int{"max_uses": 20}, true))
	for i := 1; i < 20; i++ {
		if _, err := clients[i].request("POST", "/invites/"+invite.Code, nil, true); err != nil {
			panic(err)
		}
	}
	channels := decode[[]entity](clients[0].request("GET", "/guilds/"+guild.ID+"/channels", nil, true))
	channel := ""
	for _, ch := range channels {
		if ch.Type == 0 {
			channel = ch.ID
			break
		}
	}
	if channel == "" {
		panic("no text channel")
	}
	path := "/channels/" + channel + "/messages"
	cursors := make([]string, 1000)
	for i := range cursors {
		msg := decode[entity](clients[i%20].request("POST", path, map[string]string{"content": fmt.Sprintf("seed-%s-%d", run, i)}, true))
		cursors[i] = msg.ID
		time.Sleep(70 * time.Millisecond)
	}
	// Refresh after setup so the first registered member's 15-minute access
	// token remains valid through the entire measured load. Setup may take minutes
	// because credential throttling is deliberately unchanged.
	for i := range clients {
		clients[i].refresh()
	}
	m := &measurement{pending: map[string]*delivery{}}
	sockets := make([]*websocket.Conn, 20)
	defer func() {
		m.mu.Lock()
		m.closed = true
		m.mu.Unlock()
		for _, ws := range sockets {
			if ws != nil {
				_ = ws.Close()
			}
		}
	}()
	for i := range sockets {
		var err error
		sockets[i], err = connect(clients[i], i, m)
		if err != nil {
			panic(err)
		}
	}
	fmt.Println("setup: channel has 1000 messages; 20 gateways READY")
	history := make([]float64, 0, 100)
	rng := mathrand.New(mathrand.NewSource(1))
	for i := 0; i < 100; i++ {
		start := time.Now()
		rows := decode[[]entity](clients[i%20].request("GET", path+"?limit=50&before="+cursors[50+rng.Intn(950)], nil, false))
		if len(rows) != 50 {
			panic(fmt.Sprintf("history page has %d rows", len(rows)))
		}
		history = append(history, float64(time.Since(start).Microseconds())/1000)
	}
	// Credential refresh already leaves more than a minute after seed writes.
	// Begin immediately after READY/history; an idle cooldown could conceal a
	// connection-lifetime defect before the measured profile even starts.
	for _, client := range clients {
		parts := strings.Split(client.token, ".")
		if len(parts) != 3 {
			panic("invalid setup access token")
		}
		raw, err := base64.RawURLEncoding.DecodeString(parts[1])
		if err != nil {
			panic("invalid setup access token")
		}
		var claims struct {
			Exp int64 `json:"exp"`
		}
		if json.Unmarshal(raw, &claims) != nil {
			panic("invalid setup access token")
		}
		if time.Until(time.Unix(claims.Exp, 0)) < *duration+time.Minute {
			panic("setup credential lifetime insufficient for measured load")
		}
	}
	fmt.Printf("load: starting %s, 20 clients, one send/client/10s; no retries\n", *duration)
	start := time.Now()
	attempts, sendErrors := 0, 0
	for tick := 0; time.Duration(tick)*10*time.Second < *duration; tick++ {
		time.Sleep(time.Until(start.Add(time.Duration(tick) * 10 * time.Second)))
		var wg sync.WaitGroup
		var errorsMu sync.Mutex
		for i := range clients {
			wg.Add(1)
			go func(i int) {
				defer wg.Done()
				key := fmt.Sprintf("load-%s-%d-%d", run, tick, i)
				m.mu.Lock()
				m.pending[key] = &delivery{sender: i, sent: time.Now(), seen: map[int]bool{}}
				m.mu.Unlock()
				_, err := clients[i].request("POST", path, map[string]string{"content": key}, false)
				if err != nil {
					errorsMu.Lock()
					sendErrors++
					errorsMu.Unlock()
				}
			}(i)
		}
		wg.Wait()
		attempts += 20
	}
	time.Sleep(time.Until(start.Add(*duration)))
	time.Sleep(5 * time.Second)
	m.mu.Lock()
	lost := attempts*19 - len(m.latencies)
	result := map[string]any{"duration_seconds": duration.Seconds(), "clients": 20, "send_attempts": attempts, "send_errors": sendErrors, "expected_deliveries": attempts * 19, "received_deliveries": len(m.latencies), "lost_deliveries": lost, "duplicates": m.duplicates, "unexpected": m.unexpected, "disconnects": m.disconnects, "gateway_p95_ms": quantile(m.latencies), "history_calls": len(history), "history_p95_ms": quantile(history), "channel_id": channel}

	loadP95 := p95(m.latencies)
	loadDisconnects, loadDuplicates, loadUnexpected := m.disconnects, m.duplicates, m.unexpected
	m.pending = map[string]*delivery{}
	m.latencies = nil
	m.mu.Unlock()
	time.Sleep(time.Minute)
	clients[0].refresh()
	// Explicit SYS-NFR-002 acceptance: one sender sends 100 messages with all
	// 20 members connected. 1.3s spacing stays below 50 messages/user/minute.
	for i := 0; i < 100; i++ {
		key := fmt.Sprintf("load-%s-single-%d", run, i)
		m.mu.Lock()
		m.pending[key] = &delivery{sender: 0, sent: time.Now(), seen: map[int]bool{}}
		m.mu.Unlock()
		if _, err := clients[0].request("POST", path, map[string]string{"content": key}, false); err != nil {
			sendErrors++
		}
		time.Sleep(1300 * time.Millisecond)
	}
	time.Sleep(5 * time.Second)
	m.mu.Lock()
	defer m.mu.Unlock()
	result["single_sender_attempts"] = 100
	result["single_sender_expected_deliveries"] = 1900
	result["single_sender_received_deliveries"] = len(m.latencies)
	result["single_sender_p95_ms"] = quantile(m.latencies)
	result["send_errors"] = sendErrors
	b, _ := json.MarshalIndent(result, "", "  ")
	fmt.Println(string(b))
	if lost != 0 || sendErrors != 0 || loadDisconnects != 0 || loadDuplicates != 0 || loadUnexpected != 0 || m.disconnects != 0 || m.duplicates != 0 || m.unexpected != 0 || len(m.latencies) != 1900 || loadP95 > 1000 || p95(m.latencies) > 1000 || p95(history) > 300 {
		os.Exit(1)
	}
}

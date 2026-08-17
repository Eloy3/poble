package server

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"poble/internal/game"

	"github.com/gorilla/websocket"
)

type Envelope struct {
	Type    string           `json:"type"`
	Text    string           `json:"text,omitempty"`
	RoomID  string           `json:"roomId,omitempty"`
	Player  string           `json:"player,omitempty"`
	State   any              `json:"state,omitempty"`
	Players []map[string]any `json:"players,omitempty"`
}

type Client struct {
	Conn     *websocket.Conn
	ID       string // websocket connection ID
	PlayerID string
}

type Hub struct {
	mu      sync.Mutex
	clients map[*Client]struct{}
	game    *game.Game
}

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

func NewHub() *Hub {
	return &Hub{
		clients: make(map[*Client]struct{}),
		game:    game.NewGame(),
	}
}

func (h *Hub) HandleWS(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("upgrade:", err)
		return
	}
	defer conn.Close()

	client := &Client{Conn: conn, ID: fmt.Sprintf("client-%d", time.Now().UnixNano())}

	h.mu.Lock()
	h.clients[client] = struct{}{}
	h.mu.Unlock()

	defer h.removeClient(client)

	for {
		_, raw, err := conn.ReadMessage()
		if err != nil {
			break
		}

		var env Envelope
		if err := json.Unmarshal(raw, &env); err != nil {
			continue
		}

		switch env.Type {
		case "join":
			h.handleJoin(client, env)
		case "chat":
			h.handleChat(client, env)
		case "start_game":
			h.handleStartGame(client)
		case "night_action":
			h.handleNightAction(client, env)
		case "resolve_night":
			h.handleResolveNight(client)
		case "ready":
			h.handleReady(client)
		default:
			h.send(client, Envelope{Type: "error", Text: "unknown message type"})
		}
	}
}

func (h *Hub) handleReady(client *Client) {
	player := h.game.FindPlayerByID(client.PlayerID)
	if player != nil && h.game.Phase == game.PhaseLobby {
		player.Ready = true
		h.game.Ready++
		h.broadcastGame(Envelope{Type: "ready", Text: fmt.Sprintf("%s is ready. (%d/%d)", player.Name, h.game.Ready, game.Limit)})
	}
}

func (h *Hub) handleJoin(client *Client, env Envelope) {
	name := strings.TrimSpace(env.Text)
	if name == "" {
		name = "Anonymous"
	}

	player := h.game.FindPlayerByName(name)
	if player == nil {
		newPlayer, err := h.game.AddPlayer(name)
		if err != nil {
			h.send(client, Envelope{Type: "error", Text: err.Error()})
			return
		}
		player = newPlayer
	}

	client.PlayerID = player.ID
	h.sendPlayerView(client, player)
	h.broadcastGame(Envelope{Type: "player_joined", Text: fmt.Sprintf("%s joined the game.", player.Name)})
}

func (h *Hub) handleChat(client *Client, env Envelope) {
	if client.PlayerID == "" {
		h.send(client, Envelope{Type: "error", Text: "join first"})
		return
	}

	msg := strings.TrimSpace(env.Text)
	if msg == "" {
		h.send(client, Envelope{Type: "error", Text: "message is empty"})
		return
	}

	player := h.game.FindPlayerByID(client.PlayerID)
	if player == nil {
		h.send(client, Envelope{Type: "error", Text: "player not found"})
		return
	}

	h.broadcastGame(Envelope{Type: "chat", Player: player.Name, Text: player.Name + ": " + msg})
}

func (h *Hub) handleStartGame(client *Client) {
	if err := h.game.StartGame(); err != nil {
		h.send(client, Envelope{Type: "error", Text: err.Error()})
		return
	}
	h.broadcastGame(Envelope{Type: "game_started", State: h.game})
}

func (h *Hub) handleNightAction(client *Client, env Envelope) {
	if h.game.Phase != game.PhaseNight {
		h.send(client, Envelope{Type: "error", Text: "game has not started"})
		return
	}
	if client.PlayerID == "" {
		h.send(client, Envelope{Type: "error", Text: "join first"})
		return
	}
	if err := h.game.ResolveNightAction(client.PlayerID, env.Text); err != nil {
		h.send(client, Envelope{Type: "error", Text: err.Error()})
		return
	}
	player := h.game.FindPlayerByID(client.PlayerID)
	h.broadcastGame(Envelope{Type: "night_action", Text: fmt.Sprintf("%s acted at night.", player.Name)})
}

func (h *Hub) handleResolveNight(client *Client) {
	if h.game.Phase != game.PhaseNight {
		h.send(client, Envelope{Type: "error", Text: "it is not night"})
		return
	}
	h.game.ResolveNight()
	h.broadcastGame(Envelope{Type: "state_update", State: h.game})
}

func (h *Hub) send(client *Client, msg Envelope) {
	payload, err := json.Marshal(msg)
	if err != nil {
		log.Println("marshal:", err)
		return
	}
	if err := client.Conn.WriteMessage(websocket.TextMessage, payload); err != nil {
		log.Println("write:", err)
	}
}

func (h *Hub) broadcastGame(msg Envelope) {
	payload, err := json.Marshal(msg)
	if err != nil {
		log.Println("marshal game:", err)
		return
	}

	h.mu.Lock()
	defer h.mu.Unlock()
	for client := range h.clients {
		if err := client.Conn.WriteMessage(websocket.TextMessage, payload); err != nil {
			log.Println("game write:", err)
		}
	}
}

func (h *Hub) sendPlayerView(client *Client, player *game.Player) {
	if player == nil {
		h.send(client, Envelope{Type: "error", Text: "player not found"})
		return
	}
	state := map[string]any{
		"playerName": player.Name,
		"role":       player.Role.Name,
		"alive":      player.Alive,
		"phase":      h.game.Phase,
		"players": func() []map[string]any {
			result := make([]map[string]any, 0, len(h.game.Players))
			for _, p := range h.game.Players {
				result = append(result, map[string]any{"name": p.Name, "alive": p.Alive, "role": p.Role.Name})
			}
			return result
		}(),
		"eventLog": h.game.Events,
	}
	h.send(client, Envelope{Type: "state", State: state})
}

func (h *Hub) removeClient(client *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()
	delete(h.clients, client)
}

func ServeHTTP() http.Handler {
	mux := http.NewServeMux()
	hub := NewHub()
	mux.HandleFunc("/ws", hub.HandleWS)
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})
	mux.Handle("/", http.FileServer(http.Dir("./web")))

	return loggingMiddleware(mux)
}

func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		log.Printf("%s %s %s", r.Method, r.URL.Path, time.Since(start))
	})
}

func GetPort() string {
	port := os.Getenv("PORT")
	if port == "" {
		return "8080"
	}
	return port
}

func setupServer() *http.Server {
	return &http.Server{
		Addr:              ":" + GetPort(),
		Handler:           ServeHTTP(),
		ReadHeaderTimeout: 5 * time.Second,
		WriteTimeout:      15 * time.Second,
		IdleTimeout:       30 * time.Second,
	}
}

func Run() error {
	server := setupServer()
	log.Printf("server listening on %s", server.Addr)
	if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return err
	}
	return nil
}

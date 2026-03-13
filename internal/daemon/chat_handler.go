package daemon

import (
	"encoding/json"
	"log/slog"
	"net"
)

// ChatRequest is a client→server message on a NDJSON chat connection.
type ChatRequest struct {
	Type      string `json:"type"`
	SessionID string `json:"session_id,omitempty"`
	Content   string `json:"content,omitempty"`
}

// ChatEvent is a server→client message on a NDJSON chat connection.
type ChatEvent struct {
	Type      string `json:"type"`
	SessionID string `json:"session_id,omitempty"`
	Content   string `json:"content,omitempty"`
	Welcome   string `json:"welcome,omitempty"`
	Name      string `json:"name,omitempty"`
	Args      string `json:"args,omitempty"`
	Message   string `json:"message,omitempty"`
}

// handleChatConnection processes a long-lived NDJSON chat connection.
// It reads requests in a loop and dispatches to the session manager.
func (s *SocketServer) handleChatConnection(conn net.Conn, firstReq *ChatRequest, sessions *SessionManager, logger *slog.Logger) {
	decoder := json.NewDecoder(conn)
	encoder := json.NewEncoder(conn)

	// Process the first request (already decoded by the dispatcher).
	s.dispatchChatRequest(firstReq, conn, encoder, sessions, logger)

	// Continue reading subsequent requests on the same connection.
	for {
		var req ChatRequest
		if err := decoder.Decode(&req); err != nil {
			return // connection closed or error
		}
		s.dispatchChatRequest(&req, conn, encoder, sessions, logger)
	}
}

func (s *SocketServer) dispatchChatRequest(req *ChatRequest, conn net.Conn, encoder *json.Encoder, sessions *SessionManager, logger *slog.Logger) {
	switch req.Type {
	case "session.create":
		sess := sessions.Create()
		logger.Info("session created", "session_id", sess.ID)
		encoder.Encode(ChatEvent{
			Type:      "session.created",
			SessionID: sess.ID,
			Welcome:   sessions.WelcomeText(),
		})

	case "session.resume":
		sess := sessions.Get(req.SessionID)
		if sess == nil {
			encoder.Encode(ChatEvent{Type: "error", Message: "session not found: " + req.SessionID})
			return
		}
		encoder.Encode(ChatEvent{
			Type:      "session.created",
			SessionID: sess.ID,
			Welcome:   sessions.WelcomeText(),
		})

	case "message":
		s.handleChatMessage(req, encoder, sessions, logger)

	case "session.close":
		logger.Info("session closed", "session_id", req.SessionID)
		sessions.Close(s.ctx, req.SessionID)
	}
}

func (s *SocketServer) handleChatMessage(req *ChatRequest, encoder *json.Encoder, sessions *SessionManager, logger *slog.Logger) {
	onChunk := func(chunk string) {
		encoder.Encode(ChatEvent{Type: "chunk", Content: chunk})
	}
	onToolCall := func(name string, args string) {
		encoder.Encode(ChatEvent{Type: "tool_call", Name: name, Args: args})
	}

	content, err := sessions.RunTurnStream(s.ctx, req.SessionID, req.Content, onChunk, onToolCall)
	if err != nil {
		logger.Error("chat turn error", "session_id", req.SessionID, "error", err)
		encoder.Encode(ChatEvent{Type: "error", Message: err.Error()})
		return
	}
	encoder.Encode(ChatEvent{Type: "done", Content: content})
}

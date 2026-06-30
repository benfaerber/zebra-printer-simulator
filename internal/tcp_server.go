package internal

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net"
	"strings"
	"time"
)

type TCPServer struct {
	addr     string
	state    *PrinterState
	renderer *Renderer
	webhook  Webhook
}

type TCPServerOptions struct {
	Addr     string
	State    *PrinterState
	Renderer *Renderer
	Webhook  Webhook
}

func NewTCPServer(opts TCPServerOptions) *TCPServer {
	if opts.Webhook == nil {
		opts.Webhook = NoopWebhook{}
	}
	return &TCPServer{
		addr:     opts.Addr,
		state:    opts.State,
		renderer: opts.Renderer,
		webhook:  opts.Webhook,
	}
}

func (s *TCPServer) Start(ctx context.Context) error {
	listener, err := net.Listen("tcp", s.addr)
	if err != nil {
		return fmt.Errorf("listen on %s: %w", s.addr, err)
	}

	slog.Info("TCP printer simulator listening", "addr", s.addr)

	go func() {
		<-ctx.Done()
		listener.Close()
	}()

	for {
		conn, err := listener.Accept()
		if err != nil {
			select {
			case <-ctx.Done():
				return nil
			default:
				slog.Warn("accept error", "err", err)
				continue
			}
		}
		go s.handleConnection(conn)
	}
}

func (s *TCPServer) handleConnection(conn net.Conn) {
	defer conn.Close()
	slog.Info("new connection", "remote", conn.RemoteAddr())

	var buffer strings.Builder

	for {
		_ = conn.SetReadDeadline(time.Now().Add(30 * time.Second))
		buf := make([]byte, 65536)
		n, err := conn.Read(buf)
		if n > 0 {
			buffer.WriteString(string(buf[:n]))
			s.processBuffer(&buffer, conn)
		}

		if err != nil {
			if err != io.EOF {
				slog.Debug("read error", "err", err)
			}
			s.flushRemainingLabels(&buffer)
			break
		}
	}
}

func (s *TCPServer) processBuffer(buffer *strings.Builder, conn net.Conn) {
	for {
		content := buffer.String()
		if len(content) == 0 {
			return
		}

		if s.tryHandleCommand(content, buffer, conn) {
			continue
		}

		label, remainder, found := extractNextLabel(content)
		if !found {
			return
		}

		s.handleZPL(label)
		buffer.Reset()
		buffer.WriteString(remainder)
	}
}

func (s *TCPServer) tryHandleCommand(content string, buffer *strings.Builder, conn net.Conn) bool {
	trimmed := strings.TrimSpace(content)
	if trimmed == "" {
		buffer.Reset()
		return true
	}

	if !strings.HasPrefix(trimmed, "~HS") && !strings.HasPrefix(trimmed, "! U1 ") {
		return false
	}

	nlIdx := strings.IndexByte(content, '\n')
	var line, remainder string
	if nlIdx >= 0 {
		line = content[:nlIdx+1]
		remainder = content[nlIdx+1:]
	} else {
		line = content
		remainder = ""
	}

	cmdType := ClassifyInput(line)
	switch cmdType {
	case CommandHS:
		s.handleHS(conn)
	case CommandSGD:
		s.handleSGD(conn, line)
	default:
		return false
	}

	buffer.Reset()
	buffer.WriteString(remainder)
	return true
}

func extractNextLabel(content string) (label string, remainder string, found bool) {
	startIdx := strings.Index(content, "^XA")
	if startIdx < 0 {
		return "", content, false
	}

	endMarker := "^XZ"
	endIdx := strings.Index(content[startIdx:], endMarker)
	if endIdx < 0 {
		return "", content, false
	}

	endPos := startIdx + endIdx + len(endMarker)
	return content[startIdx:endPos], content[endPos:], true
}

func (s *TCPServer) flushRemainingLabels(buffer *strings.Builder) {
	content := buffer.String()
	for {
		label, rest, found := extractNextLabel(content)
		if !found {
			break
		}
		s.handleZPL(label)
		content = rest
	}
}

func (s *TCPServer) handleHS(conn net.Conn) {
	response := s.state.GenerateHSResponse()
	_, err := conn.Write([]byte(response))
	if err != nil {
		slog.Warn("failed to write HS response", "err", err)
	}
	slog.Info("sent ~HS response")
}

func (s *TCPServer) handleSGD(conn net.Conn, data string) {
	response := HandleSGDCommand(data, s.state)
	_, err := conn.Write([]byte(response))
	if err != nil {
		slog.Warn("failed to write SGD response", "err", err)
	}
	slog.Info("sent SGD response", "response", strings.TrimSpace(response))
}

func (s *TCPServer) handleZPL(data string) {
	s.state.SetFormatsInBuffer(s.state.FormatsInBuffer() + 1)

	path, err := s.renderer.RenderZPL([]byte(data))
	if err != nil {
		slog.Warn("render failed", "err", err)
		s.state.SetFormatsInBuffer(s.state.FormatsInBuffer() - 1)
		s.state.IncrementRenderFailures()
		return
	}

	s.state.IncrementLabelCount()
	s.state.SetFormatsInBuffer(s.state.FormatsInBuffer() - 1)
	slog.Info("rendered label", "path", path, "label_count", s.state.LabelCount())

	s.webhook.Notify(eventFromPath(path, s.state.LabelCount()))
}

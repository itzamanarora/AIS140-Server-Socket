package server

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"
	"sync"
	"sync/atomic"
	"time"
	"unicode"

	"github.com/aman/ais140-socket/internal/config"
)

// Server is a raw TCP capture listener for AIS-140 devices.
// It accepts connections and dumps received bytes without decoding.
type Server struct {
	cfg      config.Config
	log      *slog.Logger
	listener net.Listener

	connID   atomic.Uint64
	wg       sync.WaitGroup
	shutdown atomic.Bool
}

// New creates a Server with the given config and logger.
func New(cfg config.Config, log *slog.Logger) *Server {
	return &Server{
		cfg: cfg,
		log: log,
	}
}

// StartServer binds the TCP listener and accepts connections until ctx is cancelled.
func (s *Server) StartServer(ctx context.Context) error {
	ln, err := net.Listen("tcp", s.cfg.Addr())
	if err != nil {
		return fmt.Errorf("listen on %s: %w", s.cfg.Addr(), err)
	}
	s.listener = ln

	fmt.Println("---------------------------------------------------")
	fmt.Println("AIS-140 TCP Server Started")
	fmt.Printf("Listening on %s\n", s.cfg.Addr())
	fmt.Println("---------------------------------------------------")
	fmt.Println()

	s.log.Info("server listening",
		"addr", s.cfg.Addr(),
		"read_timeout", s.cfg.ReadTimeout.String(),
		"write_timeout", s.cfg.WriteTimeout.String(),
		"read_buffer", s.cfg.ReadBuffer,
	)

	acceptDone := make(chan struct{})
	go func() {
		<-ctx.Done()
		s.shutdown.Store(true)
		_ = ln.Close()
		close(acceptDone)
	}()

	for {
		conn, err := ln.Accept()
		if err != nil {
			if s.shutdown.Load() || errors.Is(err, net.ErrClosed) {
				break
			}
			s.log.Error("accept failed", "error", err)
			continue
		}

		id := s.connID.Add(1)
		s.wg.Add(1)
		go func(id uint64, c net.Conn) {
			defer s.wg.Done()
			s.HandleConnection(ctx, id, c)
		}(id, conn)
	}

	<-acceptDone
	s.wg.Wait()
	s.log.Info("server stopped")
	return nil
}

// HandleConnection manages a single client connection lifecycle.
func (s *Server) HandleConnection(ctx context.Context, id uint64, conn net.Conn) {
	defer conn.Close()

	// Unblock Read on shutdown / context cancel.
	go func() {
		<-ctx.Done()
		_ = conn.Close()
	}()

	remote := conn.RemoteAddr().String()
	connectedAt := time.Now()

	fmt.Println("[NEW CONNECTION]")
	fmt.Printf("Connection ID : %d\n", id)
	fmt.Printf("Client        : %s\n", remote)
	fmt.Printf("Connected At  : %s\n", connectedAt.Format("2006-01-02 15:04:05"))
	fmt.Println()

	s.log.Info("client connected",
		"connection_id", id,
		"remote", remote,
		"connected_at", connectedAt.Format(time.RFC3339),
	)

	err := s.ReadPackets(ctx, id, conn)

	duration := time.Since(connectedAt).Round(time.Millisecond)
	fmt.Println("---------------------------------------------------")
	fmt.Println()
	fmt.Println("[CLIENT DISCONNECTED]")
	fmt.Printf("Connection ID : %d\n", id)
	fmt.Printf("Duration      : %s\n", duration)
	fmt.Println()

	if err != nil && !errors.Is(err, io.EOF) && !errors.Is(err, net.ErrClosed) && !isTimeout(err) {
		s.log.Warn("client disconnected with error",
			"connection_id", id,
			"remote", remote,
			"duration", duration.String(),
			"error", err,
		)
		return
	}

	s.log.Info("client disconnected",
		"connection_id", id,
		"remote", remote,
		"duration", duration.String(),
	)
}

// ReadPackets continuously reads raw bytes from conn until disconnect or context cancel.
func (s *Server) ReadPackets(ctx context.Context, id uint64, conn net.Conn) error {
	buf := make([]byte, s.cfg.ReadBuffer)

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		if s.cfg.ReadTimeout > 0 {
			if err := conn.SetReadDeadline(time.Now().Add(s.cfg.ReadTimeout)); err != nil {
				return fmt.Errorf("set read deadline: %w", err)
			}
		}

		n, err := conn.Read(buf)
		if n > 0 {
			packet := make([]byte, n)
			copy(packet, buf[:n])
			s.PrintPacket(id, packet)
		}

		if err != nil {
			if isTimeout(err) {
				// Idle timeout: keep waiting unless shutting down.
				if s.shutdown.Load() {
					return err
				}
				continue
			}
			if errors.Is(err, io.EOF) {
				return io.EOF
			}
			return err
		}
	}
}

// PrintPacket dumps a raw packet in hex and ASCII without any protocol decoding.
func (s *Server) PrintPacket(id uint64, data []byte) {
	now := time.Now()

	fmt.Println("[PACKET RECEIVED]")
	fmt.Printf("Connection ID : %d\n", id)
	fmt.Printf("Timestamp : %s\n", now.Format("15:04:05.000"))
	fmt.Printf("Length    : %d bytes\n", len(data))
	fmt.Println()
	fmt.Println("HEX:")
	fmt.Println(formatHex(data))
	fmt.Println()
	fmt.Println("ASCII:")
	fmt.Println(formatASCII(data))
	fmt.Println()
	fmt.Println("---------------------------------------------------")
	fmt.Println()

	s.log.Info("packet received",
		"connection_id", id,
		"length", len(data),
		"timestamp", now.Format(time.RFC3339Nano),
	)
}

func formatHex(data []byte) string {
	const bytesPerLine = 16
	var b []byte
	for i, v := range data {
		if i > 0 && i%bytesPerLine == 0 {
			b = append(b, '\n')
		} else if i > 0 {
			b = append(b, ' ')
		}
		b = append(b, fmt.Sprintf("%02X", v)...)
	}
	return string(b)
}

func formatASCII(data []byte) string {
	out := make([]byte, len(data))
	for i, v := range data {
		r := rune(v)
		if v >= 32 && v < 127 && unicode.IsPrint(r) {
			out[i] = v
		} else {
			out[i] = '.'
		}
	}
	return string(out)
}

func isTimeout(err error) bool {
	var ne net.Error
	return errors.As(err, &ne) && ne.Timeout()
}

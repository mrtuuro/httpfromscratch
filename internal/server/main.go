package server

import (
    "fmt"
    "log"
    "net"
    "sync/atomic"
)

type Server struct {
    listener net.Listener
    closed   atomic.Bool
}

func Serve(port int) (*Server, error) {
    l, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
    if err != nil {
        return nil, err
    }

    s := NewServer(l)
    go s.listen()
    return s, nil
}
func NewServer(listener net.Listener) *Server {
    return &Server{
        listener: listener,
    }
}

func (s *Server) Close() error {
    s.closed.Store(true)
    if s.listener != nil {
        return s.listener.Close()
    }
    return nil
}

func (s *Server) listen() {
    for {
        conn, err := s.listener.Accept()
        if err != nil {
            if s.closed.Load() {
                return
            }
            log.Printf("Error accepting connection: %v", err)
            continue
        }
        go s.handle(conn)
    }
}

func (s *Server) handle(conn net.Conn) {
    defer conn.Close()
    response := fmt.Sprintf("HTTP/1.1 200 OK\nContent-Type: text/plain\n\nHello World!\n")
    conn.Write([]byte(response))
    return
}

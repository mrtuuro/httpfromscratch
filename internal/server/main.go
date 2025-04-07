package server

import (
    "fmt"
    "log"
    "net"
    "sync/atomic"

    "github.com/mrtuuro/http-from-tcp/internal/request"
    "github.com/mrtuuro/http-from-tcp/internal/response"
)

type Handler func(w *response.Writer, req *request.Request)

type Server struct {
    handler  Handler
    listener net.Listener
    closed   atomic.Bool
}

func NewServer(h Handler) *Server {
    return &Server{
        handler: h,
    }
}

func Serve(port int, handler Handler) (*Server, error) {
    s := NewServer(handler)
    l, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
    if err != nil {
        return nil, err
    }

    s.listener = l
    go s.listen()
    return s, nil
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

    w := response.NewWriter(conn)
    req, err := request.RequestFromReader(conn)
    if err != nil {
        w.WriteStatusLine(response.StatusBadRequest)
        body := []byte(fmt.Sprintf("Error parsing request: %v", err))
        w.WriteHeaders(response.GetDefaultHeaders(len(body)))
        w.WriteBody(body)
        return
    }
    s.handler(w, req)
    return
}

func (s *Server) Close() error {
    s.closed.Store(true)
    if s.listener != nil {
        return s.listener.Close()
    }
    return nil
}

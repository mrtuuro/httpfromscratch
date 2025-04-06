package server

import (
    "bytes"
    "fmt"
    "io"
    "log"
    "net"
    "sync/atomic"

    "github.com/mrtuuro/http-from-tcp/internal/request"
    "github.com/mrtuuro/http-from-tcp/internal/response"
)

type Handler func(w io.Writer, req *request.Request) *HandlerError

type HandlerError struct {
    StatusCode response.StatusCode
    Message    string
}

func (he HandlerError) Write(w io.Writer) {
    response.WriteStatusLine(w, he.StatusCode)
    messageBytes := []byte(he.Message)
    headers := response.GetDefaultHeaders(len(messageBytes))
    response.WriteHeaders(w, headers)
    w.Write(messageBytes)
}

type Server struct {
    handler  Handler
    listener net.Listener
    closed   atomic.Bool
}

func Serve(port int, handler Handler) (*Server, error) {
    l, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
    if err != nil {
        return nil, err
    }

    s := NewServer(handler, l)
    go s.listen()
    return s, nil
}
func NewServer(h Handler, listener net.Listener) *Server {
    return &Server{
        handler:  h,
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
    w := response.NewHandlerWriter(conn)

    req, err := request.RequestFromReader(conn)
    if err != nil {
        hErr := &HandlerError{
            StatusCode: response.StatusBadRequest,
            Message:    err.Error(),
        }
        hErr.Write(w)
        return
    }

    buf := bytes.NewBuffer([]byte{})
    hErr := s.handler(buf, req)
    if hErr != nil {
        hErr.Write(w)
        return
    }

    b := buf.Bytes()
    response.WriteStatusLine(w, response.StatusOK)
    hdrs := response.GetDefaultHeaders(len(b))
    if err := response.WriteHeaders(w, hdrs); err != nil {
        log.Printf("Error response: %v", err)
    }
    w.Write(b)
}

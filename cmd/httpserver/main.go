package main

import (
    "log"
    "os"
    "os/signal"
    "syscall"

    "github.com/mrtuuro/http-from-tcp/internal/request"
    "github.com/mrtuuro/http-from-tcp/internal/response"
    "github.com/mrtuuro/http-from-tcp/internal/server"
)

const port = 42069

func main() {
    server, err := server.Serve(port, ServerHandler)
    if err != nil {
        log.Fatalf("Error starting server: %v", err)
    }
    defer server.Close()
    log.Println("Server started on port", port)

    sigChan := make(chan os.Signal, 1)
    signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
    <-sigChan
    log.Println("Server gravefully stopped")
}

func ServerHandler(w *response.Writer, req *request.Request) {
    if req.RequestLine.RequestTarget == "/yourproblem" {
        handler400(w, req)
        return
    }
    if req.RequestLine.RequestTarget == "/myproblem" {
        handler500(w, req)
        return
    }
    handler200(w, req)
    return
}

func handler400(w *response.Writer, _ *request.Request) {
    w.WriteStatusLine(response.StatusBadRequest)
    body := []byte(`<html>
    <head>
    <title>400 Bad Request</title>
    </head>
    <body>
    <h1>Bad Request</h1>
    <p>Your request honestly kinda sucked.</p>
    </body>
    </html>
    `)
    h := response.GetDefaultHeaders(len(body))
    h.Override("Content-Type", "text/html")
    w.WriteHeaders(h)
    w.WriteBody(body)
    return
}

func handler500(w *response.Writer, _ *request.Request) {
    w.WriteStatusLine(response.StatusInternalServerError)
    body := []byte(`<html>
    <head>
    <title>500 Internal Server Error</title>
    </head>
    <body>
    <h1>Internal Server Error</h1>
    <p>Okay, you know what? This one is on me.</p>
    </body>
    </html>
    `)
    h := response.GetDefaultHeaders(len(body))
    h.Override("Content-Type", "text/html")
    w.WriteHeaders(h)
    w.WriteBody(body)
}

func handler200(w *response.Writer, _ *request.Request) {
    w.WriteStatusLine(response.StatusOK)
    body := []byte(`<html>
    <head>
    <title>200 OK</title>
    </head>
    <body>
    <h1>Success!</h1>
    <p>Your request was an absolute banger.</p>
    </body>
    </html>
    `)
    h := response.GetDefaultHeaders(len(body))
    h.Override("Content-Type", "text/html")
    w.WriteHeaders(h)
    w.WriteBody(body)
    return
}

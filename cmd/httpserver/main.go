package main

import (
    "fmt"
    "io"
    "log"
    "net/http"
    "os"
    "os/signal"
    "strings"
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
    if strings.HasPrefix(req.RequestLine.RequestTarget, "/httpbin") {
        proxyHandler(w, req)
        return
    }
    if req.RequestLine.RequestTarget == "/yourproblem" {
        handler200(w, req)
        return
    }
    if req.RequestLine.RequestTarget == "/myproblem" {
        handler500(w, req)
        return
    }
    handler200(w, req)
    return
}

func proxyHandler(w *response.Writer, req *request.Request) {
    target := strings.TrimPrefix(req.RequestLine.RequestTarget, "/httpbin/")
    url := "https://httpbin.org/" + target
    fmt.Println("Proxying to", url)
    resp, err := http.Get(url)
    if err != nil {
        handler500(w, req)
        return
    }
    defer resp.Body.Close()

    w.WriteStatusLine(response.StatusOK)
    h := response.GetDefaultHeaders(0)
    h.Override("Transfer-Encoding", "chunked")
    h.Del("Content-Length")
    w.WriteHeaders(h)

    const maxChunkSize = 8
    buffer := make([]byte, maxChunkSize)
    for {
        n, err := resp.Body.Read(buffer)
        fmt.Println("Read", n, "bytes", string(buffer))
        if n > 0 {
            _, err = w.WriteChunkedBody(buffer[:n])
            if err != nil {
                fmt.Println("Error writing chunked body:", err)
                break
            }
        }
        if err == io.EOF {
            break
        }
        if err != nil {
            fmt.Println("Error reading response body:", err)
            break
        }
    }
    _, err = w.WriteChunkedBodyDone()
    if err != nil {
        fmt.Println("Error writing chunked body done:", err)
    }

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

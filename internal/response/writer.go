package response

import (
    "fmt"
    "io"

    "github.com/mrtuuro/http-from-tcp/internal/headers"
)

type writerState int

const (
    writerStateStatusLine writerState = iota
    writerStateHeaders
    writerStateBody
)

type Writer struct {
    writer io.Writer
    state  writerState
}

func NewWriter(w io.Writer) *Writer {
    return &Writer{
        writer: w,
        state:  writerStateStatusLine,
    }
}

func (w *Writer) WriteStatusLine(statusCode StatusCode) error {
    if w.state != writerStateStatusLine {
        return fmt.Errorf("cannot write status line in state %d", w.state)
    }

    reasonPhraseMap := map[StatusCode]string{
        200: "HTTP/1.1 200 OK\r\n",
        400: "HTTP/1.1 400 Bad Request\r\n",
        500: "HTTP/1.1 500 Internal Server Error\r\n",
    }

    reasonPhrase, ok := reasonPhraseMap[statusCode]
    if !ok {
        reasonPhrase = "\r\n"
    }
    _, err := w.writer.Write([]byte(reasonPhrase))
    w.state = writerStateHeaders
    return err
}

func (w *Writer) WriteHeaders(headers headers.Headers) error {
    if w.state != writerStateHeaders {
        return fmt.Errorf("cannot write headers in state %d", w.state)
    }

    for k, v := range headers {
        headerData := []byte(fmt.Sprintf("%s: %s\r\n", k, v))
        _, err := w.writer.Write(headerData)
        if err != nil {
            return err
        }
    }
    _, err := w.writer.Write([]byte("\r\n"))
    w.state = writerStateBody
    return err
}

func (w *Writer) WriteBody(p []byte) (int, error) {
    if w.state != writerStateBody {
        return 0, fmt.Errorf("cannow write body in state %d", w.state)
    }
    return w.writer.Write(p)
}

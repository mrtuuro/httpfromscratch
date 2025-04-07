package response

import (
    "fmt"
    "io"

    "github.com/mrtuuro/http-from-tcp/internal/headers"
)

type state int

const (
    writerStateStatusLine state = iota
    writerStateHeaders
    writerStateBody
)

type Writer struct {
    writer io.Writer
    state  state
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

    defer func() { w.state = writerStateHeaders }()

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
    return err
}

func (w *Writer) WriteHeaders(headers headers.Headers) error {
    if w.state != writerStateHeaders {
        return fmt.Errorf("cannot write headers in state %d", w.state)
    }
    defer func() { w.state = writerStateBody }()

    for k, v := range headers {
        headerData := []byte(fmt.Sprintf("%s: %s\r\n", k, v))
        _, err := w.writer.Write(headerData)
        if err != nil {
            return err
        }
    }
    _, err := w.writer.Write([]byte("\r\n"))
    return err
}

func (w *Writer) WriteBody(p []byte) (int, error) {
    if w.state != writerStateBody {
        return 0, fmt.Errorf("cannow write body in state %d", w.state)
    }
    return w.writer.Write(p)
}

func (w *Writer) WriteChunkedBody(p []byte) (int, error) {
    if w.state != writerStateBody {
        return 0, fmt.Errorf("cannot write body in state %d", w.state)
    }
    chunkSize := len(p)

    nTotal := 0
    n, err := fmt.Fprintf(w.writer, "%x\r\n", chunkSize)
    if err != nil {
        return nTotal, err
    }
    nTotal += n

    n, err = w.writer.Write(p)
    if err != nil {
        return nTotal, err
    }
    nTotal += n

    n, err = w.writer.Write([]byte("\r\n"))
    if err != nil {
        return nTotal, err
    }
    nTotal += n
    return nTotal, nil
}

func (w *Writer) WriteChunkedBodyDone() (int, error) {
    if w.state != writerStateBody {
        return 0, fmt.Errorf("cannot write body in state %d", w.state)
    }
    n, err := w.writer.Write([]byte("0\r\n\r\n"))
    if err != nil {
        return n, err
    }
    return n, nil
}

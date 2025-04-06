package response

import (
    "fmt"
    "io"
    "log"

    "github.com/mrtuuro/http-from-tcp/internal/headers"
)

type writerState int
const (
    writeStatusLine writerState = iota
    writeHeaders
    writeBody
)

type Writer struct {
    io.Writer
    state writerState
}

func NewHandlerWriter(w io.Writer) *Writer {
    return &Writer{
        w,
        writeStatusLine,
    }
}
func (w *Writer) WriteStatusLine(statusCode StatusCode) error {
    reasonPhraseMap := map[StatusCode]string{
        200: "HTTP/1.1 200 OK\r\n",
        400: "HTTP/1.1 400 Bad Request\r\n",
        500: "HTTP/1.1 500 Internal Server Error\r\n",
    }

    reasonPhrase, ok := reasonPhraseMap[statusCode]
    if !ok {
        reasonPhrase = "\r\n"
    }
    _, err := w.Write([]byte(reasonPhrase))
    if err != nil {
        log.Printf("Error writing reason phrase: %v", err)
        return err
    }
    return nil
}

func (w *Writer) WriteHeaders(headers headers.Headers) error {
    for k, v := range headers {
        headerData := []byte(fmt.Sprintf("%s: %s\r\n", k, v))
        _, err := w.Write(headerData)
        if err != nil {
            return err
        }
    }
    _, err := w.Write([]byte("\r\n"))
    if err != nil {
        return err
    }
    return nil
}

func (w *Writer) WriteBody(p []byte) (int, error) {
    return 0, nil
}

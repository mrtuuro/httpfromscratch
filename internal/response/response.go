package response

import (
    "fmt"
    "io"
    "log"
    "strconv"

    "github.com/mrtuuro/http-from-tcp/internal/headers"
)

type StatusCode int

const (
    StatusOK                  StatusCode = 200
    StatusBadRequest          StatusCode = 400
    StatusInternalServerError StatusCode = 500
)

func WriteStatusLine(w io.Writer, statusCode StatusCode) error {
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

func WriteHeaders(w io.Writer, headers headers.Headers) error {
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

func GetDefaultHeaders(contentLen int) headers.Headers {
    defHeaders := headers.NewHeaders()
    defHeaders.Set("Content-Length", strconv.Itoa(contentLen))
    defHeaders.Set("Connection", "close")
    defHeaders.Set("Content-Type", "text/plain")
    return defHeaders
}

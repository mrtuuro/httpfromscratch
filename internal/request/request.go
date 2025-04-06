package request

import (
    "bytes"
    "errors"
    "fmt"
    "io"
    "strconv"
    "strings"

    "github.com/mrtuuro/http-from-tcp/internal/headers"
)

type Request struct {
    RequestLine RequestLine
    Headers     headers.Headers
    Body        []byte

    state requestState
    bodyLengthRead int
}

type RequestLine struct {
    HttpVersion   string
    RequestTarget string
    Method        string
}

type requestState int

const (
    requestStateInitialized requestState = iota
    requestStateParsingHeaders
    requestStateParsingBody
    requestStateDone
)

const crlf = "\r\n"
const bufferSize = 8

func RequestFromReader(reader io.Reader) (*Request, error) {
    buf := make([]byte, bufferSize, bufferSize)
    readToIndex := 0
    req := &Request{
        state:   requestStateInitialized,
        Headers: headers.NewHeaders(),
        Body: make([]byte, 0),
    }
    for req.state != requestStateDone {

        // NOTE: Read into buffer
        numBytesRead, err := reader.Read(buf[readToIndex:])
        if err != nil {
            if errors.Is(io.EOF, err) {
                if req.state != requestStateDone {
                    return nil, fmt.Errorf("incomplete request, in state: %d, read n bytes on EOF: %d", req.state, numBytesRead)
                }
                break
            }
            return nil, err
        }
        readToIndex += numBytesRead
        if readToIndex >= len(buf) {
            newBuf := make([]byte, len(buf)*2)
            copy(newBuf, buf)
            buf = newBuf
        }

        // NOTE: Parse from buffer
        numBytesParsed, err := req.parse(buf[:readToIndex])
        if err != nil {
            return nil, err
        }

        copy(buf, buf[numBytesParsed:])
        readToIndex -= numBytesParsed
    }
    return req, nil
}

func parseRequestLine(data []byte) (*RequestLine, int, error) {
    idx := bytes.Index(data, []byte(crlf))
    if idx == -1 {
        return nil, 0, nil
    }
    requestLineText := string(data[:idx])
    requestLine, err := requestLineFromString(requestLineText)
    if err != nil {
        return nil, 0, err
    }
    return requestLine, idx + 2, nil
}

func requestLineFromString(str string) (*RequestLine, error) {
    parts := strings.Split(str, " ")
    if len(parts) != 3 {
        return nil, fmt.Errorf("poorly formatted request-line: %s", str)
    }

    method := parts[0]
    for _, c := range method {
        if c < 'A' || c > 'Z' {
            return nil, fmt.Errorf("invalid method: %s", method)
        }
    }

    requestTarget := parts[1]

    versionParts := strings.Split(parts[2], "/")
    if len(versionParts) != 2 {
        return nil, fmt.Errorf("malformed start-line: %s", str)
    }

    httpPart := versionParts[0]
    if httpPart != "HTTP" {
        return nil, fmt.Errorf("unrecognized HTTP-version: %s", httpPart)
    }
    version := versionParts[1]
    if version != "1.1" {
        return nil, fmt.Errorf("unrecognized HTTP-version: %s", version)
    }

    return &RequestLine{
        Method:        method,
        RequestTarget: requestTarget,
        HttpVersion:   versionParts[1],
    }, nil
}

func (r *Request) parse(data []byte) (int, error) {
    totalBytesParsed := 0
    for r.state != requestStateDone {
        n, err := r.parseSingle(data[totalBytesParsed:])
        if err != nil {
            return 0, err
        }
        totalBytesParsed += n
        if n == 0 {
            break
        }
    }
    return totalBytesParsed, nil
}

func (r *Request) parseSingle(data []byte) (int, error) {
    switch r.state {
    case requestStateInitialized:
        requestLine, n, err := parseRequestLine(data)
        if err != nil {
            // something actually went wrong
            return 0, err
        }
        if n == 0 {
            // just need more data
            return 0, nil
        }
        r.RequestLine = *requestLine
        r.state = requestStateParsingHeaders
        return n, nil
    case requestStateParsingHeaders:
        n, done, err := r.Headers.Parse(data)
        if err != nil {
            return 0, err
        }
        if done {
            r.state = requestStateParsingBody
        }
        return n, nil
    case requestStateParsingBody:
        contentLen, found := r.Headers.Get([]byte("Content-Length"))
        if !found {
            r.state = requestStateDone
            return len(data), nil
        }


        contentLenStr := string(contentLen)
        contentLenInt, err := strconv.Atoi(contentLenStr)
        if err != nil {
            return 0, fmt.Errorf("error: malformed Content-Length: %s", err)
        }

        r.Body = append(r.Body, data...)
        r.bodyLengthRead += len(data)
        if r.bodyLengthRead > contentLenInt {
            return 0, fmt.Errorf("error: content-len is not equal to body length")
        }

        if r.bodyLengthRead == contentLenInt {
            r.state = requestStateDone
            return len(r.Body), nil
        }
        return len(data), nil

    case requestStateDone:
        return 0, fmt.Errorf("error: trying to read data in a done state")
    default:
        return 0, fmt.Errorf("unknown state")
    }
}

package request

import (
    "io"
    "testing"

    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
)

func TestRequestLineParse(t *testing.T) {
    // TEST: Good GET Request line
    reader := &chunkReader{
        data:            "GET / HTTP/1.1\r\nHost: localhost:42069\r\nUser-Agent: curl/7.81.0\r\nAccept: */*\r\n\r\n",
        numBytesPerRead: 3,
    }
    r, err := RequestFromReader(reader)
    require.NoError(t, err)
    require.NotNil(t, r)
    assert.Equal(t, "GET", r.RequestLine.Method)
    assert.Equal(t, "/", r.RequestLine.RequestTarget)
    assert.Equal(t, "1.1", r.RequestLine.HttpVersion)

    // TEST: Good GET Request line with path
    reader = &chunkReader{
        data:            "GET /coffee HTTP/1.1\r\nHost: localhost:42069\r\nUser-Agent: curl/7.81.0\r\nAccept: */*\r\n\r\n",
        numBytesPerRead: 1,
    }
    r, err = RequestFromReader(reader)
    require.NoError(t, err)
    require.NotNil(t, r)
    assert.Equal(t, "GET", r.RequestLine.Method)
    assert.Equal(t, "/coffee", r.RequestLine.RequestTarget)
    assert.Equal(t, "1.1", r.RequestLine.HttpVersion)

    // TEST: Good POST Request with path
    reader = &chunkReader{
        data:            "POST /coffee HTTP/1.1\r\nHost: localhost:42069\r\nUser-Agent: curl/7.81.0\r\nAccept: */*\r\n\r\n",
        numBytesPerRead: 5,
    }
    r, err = RequestFromReader(reader)
    require.NoError(t, err)
    require.NotNil(t, r)
    assert.Equal(t, "POST", r.RequestLine.Method)
    assert.Equal(t, "/coffee", r.RequestLine.RequestTarget)
    assert.Equal(t, "1.1", r.RequestLine.HttpVersion)

    // TEST: Invalid number of parts in request line
    reader = &chunkReader{
        data:            "/coffee HTTP/1.1\r\nHost: localhost:42069\r\nUser-Agent: curl/7.81.0\r\nAccept: */*\r\n\r\n",
        numBytesPerRead: 3,
    }
    r, err = RequestFromReader(reader)
    require.Error(t, err)

    // TEST: Invalid method (out of order) Request line
    reader = &chunkReader{
        data:            "/coffee POST HTTP/1.1\r\nHost: localhost:42069\r\nUser-Agent: curl/7.81.0\r\nAccept: */*\r\n\r\n",
        numBytesPerRead: 3,
    }
    r, err = RequestFromReader(reader)
    require.Error(t, err)

    // TEST: Invalid version in Request line
    reader = &chunkReader{
        data:            "OPTIONS /prime/rib TCP/1.1\r\nHost: localhost:42069\r\nUser-Agent: curl/7.81.0\r\nAccept: */*\r\n\r\n",
        numBytesPerRead: 50,
    }
    r, err = RequestFromReader(reader)
    require.Error(t, err)
}

func TestHeadersParse(t *testing.T) {
    // TEST: Standard Headers
    reader := &chunkReader{
        data:            "GET / HTTP/1.1\r\nHost: localhost:42069\r\nUser-Agent: curl/7.81.0\r\nAccept: */*\r\n\r\n",
        numBytesPerRead: 3,
    }
    r, err := RequestFromReader(reader)
    require.NoError(t, err)
    require.NotNil(t, r)
    assert.Equal(t, "localhost:42069", r.Headers["host"])
    assert.Equal(t, "curl/7.81.0", r.Headers["user-agent"])
    assert.Equal(t, "*/*", r.Headers["accept"])

    // TEST: Empty Headers
    reader = &chunkReader{
        data:            "GET / HTTP/1.1\r\n\r\n",
        numBytesPerRead: 3,
    }
    r, err = RequestFromReader(reader)
    require.NoError(t, err)
    require.NotNil(t, r)
    assert.Empty(t, r.Headers)

    // TEST: Malformed Header
    reader = &chunkReader{
        data:            "GET / HTTP/1.1\r\nHost localhost:42069\r\n\r\n",
        numBytesPerRead: 3,
    }
    r, err = RequestFromReader(reader)
    require.Error(t, err)

    // TEST: Duplicate Headers
    reader = &chunkReader{
        data:            "GET / HTTP/1.1\r\nHost: localhost:42069\r\nHost: duplicate:8080\r\n\r\n",
        numBytesPerRead: 3,
    }
    r, err = RequestFromReader(reader)
    require.NoError(t, err)
    require.NotNil(t, r)
    assert.Equal(t, "localhost:42069, duplicate:8080", r.Headers["host"])

    // TEST: Case Insensitive Headers
    reader = &chunkReader{
        data:            "GET / HTTP/1.1\r\nHOST: localhost:42069\r\nUSER-AGENT: curl/7.81.0\r\n\r\n",
        numBytesPerRead: 3,
    }
    r, err = RequestFromReader(reader)
    require.NoError(t, err)
    require.NotNil(t, r)
    assert.Equal(t, "localhost:42069", r.Headers["host"])
    assert.Equal(t, "curl/7.81.0", r.Headers["user-agent"])

    // TEST: Missing End of Headers
    reader = &chunkReader{
        data:            "GET / HTTP/1.1\r\nHost: localhost:42069\r\nUser-Agent: curl/7.81.0",
        numBytesPerRead: 3,
    }
    r, err = RequestFromReader(reader)
    require.Error(t, err)
}


func TestBodyParsing(t *testing.T) {
    // TEST: Standard Body
    reader := &chunkReader{
        data: "POST /submit HTTP/1.1\r\n" + 
        "Host: localhost:42069\r\n" + 
        "Content-Length: 13\r\n" +
        "\r\n" +
        "hello world!\n",
        numBytesPerRead: 3,
    }
    r, err := RequestFromReader(reader)
    require.NoError(t, err)
    require.NotNil(t, r)
    assert.Equal(t, "hello world!\n", string(r.Body))

    // TEST: Empty Body, 0 reported content length(valid)
    reader = &chunkReader{
        data: "POST /submit HTTP/1.1\r\n" + 
        "Host: localhost:42069\r\n" + 
        "Content-Length: 0\r\n" +
        "\r\n",
        numBytesPerRead: 3,
    }
    r, err = RequestFromReader(reader)
    require.NoError(t, err)
    require.NotNil(t, r)
    assert.Equal(t, "", string(r.Body))

    // TEST: Empty Body, no reported content length (valid)
    reader = &chunkReader{
        data: "POST /submit HTTP/1.1\r\n" + 
        "Host: localhost:42069\r\n" + 
        "\r\n",
        numBytesPerRead: 3,
    }
    r, err = RequestFromReader(reader)
    require.NoError(t, err)
    require.NotNil(t, r)



    // TEST: Body shorter than reported content length
    reader = &chunkReader{
        data: "POST /submit HTTP/1.1\r\n" + 
        "Host: localhost:42069\r\n" + 
        "Content-Length: 30\r\n" +
        "\r\n" +
        "some body\n",
        numBytesPerRead: 3,
    }
    r, err = RequestFromReader(reader)
    require.Error(t, err)

    // TEST: No Content-Length but Body Exists (valid)
    reader = &chunkReader{
        data: "POST /submit HTTP/1.1\r\n" + 
        "Host: localhost:42069\r\n" + 
        "\r\n" +
        "Video Transfer Protocol!\n",
        numBytesPerRead: 3,
    }
    r, err = RequestFromReader(reader)
    require.NoError(t, err)
    require.NotNil(t, r)
    assert.Equal(t, "", string(r.Body))
}


type chunkReader struct {
    data            string
    numBytesPerRead int
    pos             int
}

func (cr *chunkReader) Read(p []byte) (n int, err error) {
    if cr.pos >= len(cr.data) {
        return 0, io.EOF
    }
    endIndex := cr.pos + cr.numBytesPerRead
    if endIndex > len(cr.data) {
        endIndex = len(cr.data)
    }
    n = copy(p, cr.data[cr.pos:endIndex])
    cr.pos += n
    if n > cr.numBytesPerRead {
        n = cr.numBytesPerRead
        cr.pos -= n - cr.numBytesPerRead
    }
    return n, nil
}


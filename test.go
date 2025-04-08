package main

import (
    "errors"
    "fmt"
    "io"
    "net/http"
    "os"
    "path/filepath"
    "strconv"
    "time"
)

// HTML page with upload and download UI, progress bars, and ETA display.
const indexHTML = `<!DOCTYPE html>
    <html lang="en">
    <head>
    <meta charset="UTF-8">
    <title>Video Upload & Download with ETA</title>
    <style>
    body { font-family: Arial, sans-serif; margin: 40px; }
    .progress-container { margin: 10px 0; }
    .progress-bar { width: 0%; height: 20px; background: #4caf50; }
    .progress { width: 100%; background: #ddd; }
    </style>
    </head>
    <body>
    <h1>Video Upload</h1>
    <input type="file" id="uploadFile" accept="video/*">
    <button onclick="uploadFile()">Upload</button>
    <div class="progress-container">
    <div>Upload Progress:</div>
    <div class="progress"><div id="uploadProgress" class="progress-bar"></div></div>
    <div id="uploadETA"></div>
    </div>
    <div id="uploadStatus"></div>

    <h1>Video Download</h1>
    <button onclick="downloadFile()">Download Video</button>
    <div class="progress-container">
    <div>Download Progress:</div>
    <div class="progress"><div id="downloadProgress" class="progress-bar"></div></div>
    <div id="downloadETA"></div>
    </div>
    <div id="downloadStatus"></div>

    <script>
    function formatTime(seconds) {
    var hrs   = Math.floor(seconds / 3600);
    var mins  = Math.floor((seconds % 3600) / 60);
    var secs  = Math.floor(seconds % 60);
    return (hrs > 0 ? hrs + "h " : "") + (mins > 0 ? mins + "m " : "") + secs + "s";
    }

    function uploadFile() {
    var fileInput = document.getElementById("uploadFile");
    if (fileInput.files.length === 0) {
    alert("Please select a video file to upload.");
    return;
    }
    var file = fileInput.files[0];
    var formData = new FormData();
    formData.append("file", file);

    var xhr = new XMLHttpRequest();
    xhr.open("POST", "/upload", true);

    // Record the start time.
    var startTime = Date.now();

    // Update progress
    xhr.upload.onprogress = function(event) {
    if (event.lengthComputable) {
    var percentComplete = (event.loaded / event.total) * 100;
    document.getElementById("uploadProgress").style.width = percentComplete + "%";

    var elapsed = (Date.now() - startTime) / 1000; // seconds
    var rate = event.loaded / elapsed; // bytes per second
    var remainingBytes = event.total - event.loaded;
    var eta = rate > 0 ? remainingBytes / rate : 0;
    document.getElementById("uploadETA").innerText = "Estimated time remaining: " + formatTime(eta);
    }
    };

    xhr.onload = function() {
    if (xhr.status === 201) {
    document.getElementById("uploadStatus").innerText = "Upload successful: " + xhr.responseText;
    } else {
    document.getElementById("uploadStatus").innerText = "Upload failed: " + xhr.status;
    }
    };

    xhr.onerror = function() {
    document.getElementById("uploadStatus").innerText = "Upload error.";
    };

    xhr.send(formData);
    }

    function downloadFile() {
    var xhr = new XMLHttpRequest();
    xhr.open("GET", "/download", true);
    xhr.responseType = "blob";

    var startTime = Date.now();

    xhr.onprogress = function(event) {
    if (event.lengthComputable) {
    var percentComplete = (event.loaded / event.total) * 100;
    document.getElementById("downloadProgress").style.width = percentComplete + "%";

    var elapsed = (Date.now() - startTime) / 1000;
    var rate = event.loaded / elapsed;
    var remainingBytes = event.total - event.loaded;
    var eta = rate > 0 ? remainingBytes / rate : 0;
    document.getElementById("downloadETA").innerText = "Estimated time remaining: " + formatTime(eta);
    }
    };

    xhr.onload = function() {
    if (xhr.status === 200) {
    var blob = xhr.response;
    var link = document.createElement("a");
    link.href = window.URL.createObjectURL(blob);
    link.download = "downloaded_video.mp4";
    link.click();
    document.getElementById("downloadStatus").innerText = "Download complete.";
    } else {
    document.getElementById("downloadStatus").innerText = "Download failed: " + xhr.status;
    }
    };

    xhr.onerror = function() {
    document.getElementById("downloadStatus").innerText = "Download error.";
    };

    xhr.send();
    }
    </script>
    </body>
    </html>`

// uploadHandler handles video uploads.
func uploadHandler(w http.ResponseWriter, r *http.Request) {
    // Limit request body size (e.g., 100GB)
    r.Body = http.MaxBytesReader(w, r.Body, 100<<30)

    if err := r.ParseMultipartForm(10 << 20); err != nil {
        http.Error(w, "Error parsing multipart form: "+err.Error(), http.StatusBadRequest)
        return
    }

    file, fileHeader, err := r.FormFile("file")
    if err != nil {
        http.Error(w, "Error retrieving file: "+err.Error(), http.StatusBadRequest)
        return
    }
    defer file.Close()

    uploadDir := "/Users/tuuro/Records"
    if err := os.MkdirAll(uploadDir, os.ModePerm); err != nil {
        http.Error(w, "Unable to create upload directory: "+err.Error(), http.StatusInternalServerError)
        return
    }

    dstPath := filepath.Join(uploadDir, fileHeader.Filename)
    dstFile, err := os.Create(dstPath)
    if err != nil {
        http.Error(w, "Error creating destination file: "+err.Error(), http.StatusInternalServerError)
        return
    }
    defer dstFile.Close()

    progressR := newProgressReader(file, fileHeader.Size)
    if _, err := customCopy(dstFile, progressR, nil); err != nil {
        http.Error(w, "Error saving file: "+err.Error(), http.StatusInternalServerError)
        return
    }

    w.WriteHeader(http.StatusCreated)
    fmt.Fprintf(w, "File %s uploaded successfully", fileHeader.Filename)
}

func customCopy(dst io.Writer, src io.Reader, buf []byte) (written int64, err error) {
    if buf == nil {
        size := 1 * (10 << 30)
        buf = make([]byte, size)
    }
    for {
        nr, er := src.Read(buf)
        if nr > 0 {
            nw, ew := dst.Write(buf[0:nr])
            if nw < 0 || nr < nw {
                nw = 0
                if ew == nil {
                    ew = errors.New("errInvalidWrite")
                }
            }
            written += int64(nw)
            if ew != nil {
                err = ew
                break
            }
            if nr != nw {
                err = errors.New("errShortWrite")
                break
            }
        }
        if er != nil {
            if er != io.EOF {
                err = er
            }
            break
        }
    }
    return written, err
}

// downloadHandler serves the uploaded video file (first file in uploads directory).
func downloadHandler(w http.ResponseWriter, r *http.Request) {
    uploadDir := "uploads"
    files, err := os.ReadDir(uploadDir)
    if err != nil || len(files) == 0 {
        http.Error(w, "No video available for download", http.StatusNotFound)
        return
    }

    fileName := files[0].Name()
    filePath := filepath.Join(uploadDir, fileName)
    file, err := os.Open(filePath)
    if err != nil {
        http.Error(w, "Error opening file: "+err.Error(), http.StatusInternalServerError)
        return
    }
    defer file.Close()

    fi, err := file.Stat()
    if err != nil {
        http.Error(w, "Error retrieving file info: "+err.Error(), http.StatusInternalServerError)
        return
    }

    w.Header().Set("Content-Type", "video/mp4")
    w.Header().Set("Content-Disposition", "attachment; filename="+strconv.Quote(fileName))
    w.Header().Set("Content-Length", strconv.FormatInt(fi.Size(), 10))
    io.Copy(w, file)
}

// progressReader wraps an io.Reader to track progress.
type progressReader struct {
    reader   io.Reader
    total    int64
    read     int64
    lastTime time.Time
}

func newProgressReader(r io.Reader, total int64) *progressReader {
    return &progressReader{
        reader:   r,
        total:    total,
        lastTime: time.Now(),
    }
}

func (pr *progressReader) Read(p []byte) (int, error) {
    n, err := pr.reader.Read(p)
    pr.read += int64(n)
    if time.Since(pr.lastTime) >= time.Second || pr.read == pr.total {
        percent := float64(pr.read) / float64(pr.total) * 100
        fmt.Printf("Upload progress: %d/%d bytes (%.2f%%)\n", pr.read, pr.total, percent)
        pr.lastTime = time.Now()
    }
    return n, err
}

// indexHandler serves the main HTML page.
func indexHandler(w http.ResponseWriter, r *http.Request) {
    w.Header().Set("Content-Type", "text/html")
    fmt.Fprint(w, indexHTML)
}

func main() {
    http.HandleFunc("/", indexHandler)
    http.HandleFunc("/upload", uploadHandler)
    http.HandleFunc("/download", downloadHandler)
    fmt.Println("Server listening on port 8080...")
    if err := http.ListenAndServe(":8080", nil); err != nil {
        fmt.Println("Server error:", err)
    }
}

package main

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"
)

type UploadSession struct {
	FileName       string
	FilePath       string
	FileSize       int64
	UploadedChunks int
	TotalChunks    int
	Paused         bool
	Terminated     bool
	Completed      bool
	mu             sync.Mutex
}

var uploadSessions = make(map[string]*UploadSession)
var sessionMu sync.Mutex 

const chunkSize = 100

func main() {
	http.HandleFunc("/upload", handleUpload)
	http.HandleFunc("/upload/pause", handlePause)
	http.HandleFunc("/upload/status", handleUploadStatus)
	http.HandleFunc("/upload/resume", handleResume)
	fmt.Println("Starting server on port 8080")
	http.ListenAndServe(":8080", nil)
}

func handleUpload(w http.ResponseWriter, r *http.Request) {
	file, header, err := r.FormFile("file")
	if err != nil {
		http.Error(w, "File upload error", http.StatusBadRequest)
		return
	}

	fileSize, err := file.Seek(0, io.SeekEnd)
	if err != nil {
		http.Error(w, "Failed to get file size", http.StatusInternalServerError)
		return
	}
	_, _ = file.Seek(0, io.SeekStart)

	sessionID := generateSessionID()
	session := &UploadSession{
		FileName:       header.Filename,
		FilePath:       filepath.Join("uploads", header.Filename),
		FileSize:       fileSize,
		UploadedChunks: 0,
		TotalChunks:    int(fileSize / chunkSize),
	}

	sessionMu.Lock()
	uploadSessions[sessionID] = session 
	sessionMu.Unlock()

	go uploadFileInBackground(file, session)

	w.Write([]byte(fmt.Sprintf(`{"sessionID": "%s"}`, sessionID)))
}

func uploadFileInBackground(file io.Reader, session *UploadSession) {
    if session.Completed {
        return
    }
    outFile, err := os.OpenFile(session.FilePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
    if err != nil {
        fmt.Println("Error creating/opening file:", err)
        return
    }
    defer outFile.Close()

    buf := make([]byte, chunkSize)

    session.mu.Lock()
    session.Completed = false
    session.mu.Unlock()

    for {
        session.mu.Lock()
        if session.Paused || session.Terminated {
            session.mu.Unlock()
            return
        }
        session.mu.Unlock()

        n, err := file.Read(buf)
        if err != nil {
            if err == io.EOF {
				fmt.Println("break of file")
                break
            }
            fmt.Println("Error reading file chunk:", err)
            return
        }

        if n == 0 {
            fmt.Println("No bytes read, breaking out of loop.")
            break
        }

        _, err = outFile.Write(buf[:n])
        if err != nil {
            fmt.Println("Error writing chunk:", err)
            return
        }

        session.mu.Lock()
        session.UploadedChunks++ 

        if session.UploadedChunks >= session.TotalChunks {
            session.Completed = true 
            fmt.Println("Upload completed successfully.")
        }
        session.mu.Unlock()

        session.mu.Lock()
        if session.Paused {
            session.mu.Unlock()
            return
        }
        session.mu.Unlock()
    }

    session.mu.Lock()
    session.Completed = true
    session.mu.Unlock()
}

func handlePause(w http.ResponseWriter, r *http.Request) {
	sessionID := r.URL.Query().Get("sessionID")
	session, exists := getSession(sessionID)
	if !exists {
		http.Error(w, "Session not found", http.StatusNotFound)
		return
	}

	session.mu.Lock()          
	session.Paused = true       
	session.mu.Unlock()        

	w.WriteHeader(http.StatusOK)
	fmt.Fprintln(w, "Upload paused")
}

func handleResume(w http.ResponseWriter, r *http.Request) {
    sessionID := r.URL.Query().Get("sessionID")
    session, exists := getSession(sessionID)
    if !exists {
        http.Error(w, "Session not found", http.StatusNotFound)
        return
    }

    session.mu.Lock()
    if session.Completed || !session.Paused {
        session.mu.Unlock()
        http.Error(w, "Upload cannot be resumed", http.StatusConflict)
        return
    }
    session.Paused = false
    session.mu.Unlock()

    file, _, err := r.FormFile("file")
	if err != nil {
		http.Error(w, "File upload error", http.StatusBadRequest)
		return
	}

    _, err = file.Seek(int64(session.UploadedChunks*chunkSize), io.SeekStart)
    if err != nil {
        file.Close() 
        http.Error(w, "Failed to seek file", http.StatusInternalServerError)
        return
    }

    go uploadFileInBackground(file, session)

    w.WriteHeader(http.StatusOK)
    fmt.Fprintln(w, "Upload resumed")
}



func handleUploadStatus(w http.ResponseWriter, r *http.Request) {
	sessionID := r.URL.Query().Get("sessionID")
	session, exists := getSession(sessionID)
	if !exists {
		http.Error(w, "Session not found", http.StatusNotFound)
		return
	}

	session.mu.Lock() 
	status := struct {
		UploadedChunks int  `json:"uploadedChunks"`
		TotalChunks    int  `json:"totalChunks"`
		Paused         bool `json:"paused"`
		Completed      bool `json:"completed"`
	}{
		UploadedChunks: session.UploadedChunks,
		TotalChunks:    session.TotalChunks,
		Paused:         session.Paused,
		Completed:      session.Completed,
	}
	session.mu.Unlock() 

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, `{"uploadedChunks": %d, "totalChunks": %d, "paused": %t, "completed": %t}`, status.UploadedChunks, status.TotalChunks, status.Paused, status.Completed)
}

func getSession(sessionID string) (*UploadSession, bool) {
	sessionMu.Lock()
	defer sessionMu.Unlock()
	session, exists := uploadSessions[sessionID]
	return session, exists
}

func generateSessionID() string {
	return fmt.Sprintf("%d", time.Now().UnixNano())
}

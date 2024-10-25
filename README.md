# Resumable File Upload Server

A Go-based HTTP server that supports resumable file uploads with pause and resume capabilities. This server allows for reliable file uploads with progress tracking and upload state management.

## Features

- Chunk-based file uploads
- Pause and resume upload functionality
- Real-time upload status tracking
- Concurrent upload session management
- Simple HTTP API interface

## API Endpoints

### Upload File
```
POST /upload
Content-Type: multipart/form-data
```
Initiates a new file upload session. Returns a session ID that can be used to manage the upload.

**Request Body:**
- `file`: The file to upload (form-data)

**Response:**
```json
{
    "sessionID": "1234567890"
}
```

### Pause Upload
```
GET /upload/pause?sessionID={sessionID}
```
Pauses an ongoing upload session.

### Resume Upload
```
POST /upload/resume?sessionID={sessionID}
Content-Type: multipart/form-data
```
Resumes a paused upload session.

**Request Body:**
- `file`: The same file, will be seeked to the correct position automatically

### Check Upload Status
```
GET /upload/status?sessionID={sessionID}
```
Returns the current status of an upload session.

**Response:**
```json
{
    "uploadedChunks": 50,
    "totalChunks": 100,
    "paused": false,
    "completed": false
}
```

## Configuration

The server uses the following default settings:
- Port: 8080
- Chunk size: 1MB
- Upload directory: `uploads/`

## Installation

1. Clone the repository
2. Ensure Go is installed on your system
3. Create an `uploads` directory in the project root
4. Run the server:
```bash
go run main.go
```

## Implementation Details

### Upload Session Management
- Each upload is managed through an `UploadSession` struct that tracks:
  - File metadata (name, size, path)
  - Upload progress (chunks uploaded, total chunks)
  - Upload state (paused, completed, terminated)
- Sessions are stored in memory using a thread-safe map
- Concurrent access is handled using mutex locks

### File Processing
- Files are uploaded in chunks of 1 MB
- Progress is tracked per chunk
- Uploads can be paused and resumed without data loss
- Background processing ensures non-blocking operation

## Error Handling

The server handles various error conditions:
- Invalid session IDs
- File access errors
- Concurrent access conflicts
- Invalid upload states

## Limitations

- In-memory session storage (sessions are lost on server restart)
- Fixed chunk size
- No built-in authentication/authorization
- No automatic cleanup of incomplete uploads

## Future Improvements

Potential areas for enhancement:
1. Persistent session storage
2. Configurable chunk size
3. Authentication system
4. Automatic cleanup of abandoned uploads
5. Upload expiration timeouts
6. Multiple file upload support
7. Progress webhooks
8. Upload speed throttling
9. File integrity verification

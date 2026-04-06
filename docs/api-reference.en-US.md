# API Documentation

## Unified Response Format

- Successful Response:

```json
{
  "data": any,
  "requestId": string,
  "timeStamp": number
}
```

- Error Response:

```json
{
  "code": number,
  "msg": string,
  "data": any,
  "requestId": string,
  "timeStamp": number
}
```

## test Module

1. **testPing** Test Endpoint
   - HTTP Method: GET
   - Request Path: /api/v1/test/testPing
   - Request Parameters: None
   - Response Example:
   ```json
   {
     "data": {
       "time": "2025-09-26T01:46:57+08:00",
       "message": "Pong successfully!"
     },
     "requestId": "01d01617-cb23-46ec-85f1-777eeba3377c",
     "timeStamp": 1758822417
   }
   ```
2. **testHello** Test Endpoint
   - HTTP Method: GET
   - Request Path: /api/v1/test/testHello
   - Request Parameters: None
   - Response Example:
   ```json
   {
     "data": {
       "version": "1.0.0",
       "message": "Hello, magnet2video! 🎉!"
     },
     "requestId": "b42eb8af-b48d-48cd-8c15-f3cd52860d11",
     "timeStamp": 1758822421
   }
   ```
3. **testLogger** Test Endpoint
   - HTTP Method: GET
   - Request Path: /api/v1/test/testLogger
   - Request Parameters: None
   - Response Example:
   ```json
   {
     "data": {
       "level": "info",
       "message": "Log test succeeded!"
     },
     "requestId": "a74cfa1d-c313-45c4-bc1d-0a0c998d3e60",
     "timeStamp": 1758822424
   }
   ```
4. **testRedis** Test Endpoint
   - HTTP Method: POST
   - Request Path: /api/v1/test/testRedis
   - Request Parameters:
   ```json
   {
     "key": "test",
     "value": "hello",
     "ttl": 60
   }
   ```
   - Response Example:
   ```json
   {
     "data": {
       "key": "test",
       "value": "hello",
       "ttl": 60,
       "message": "Cache functionality test completed!"
     },
     "requestId": "XtZvqFlDtpgzwEAesJpFMGgJQRbQDXyM",
     "timeStamp": 1740118491
   }
   ```
5. **testSuccessRes** Test Endpoint
   - HTTP Method: GET
   - Request Path: /api/v1/test/testSuccessRes
   - Request Parameters: None
   - Response Example:
   ```json
   {
     "data": {
       "status": "success",
       "message": "Successful response validation passed!"
     },
     "requestId": "7f114931-51bc-47d5-922f-208ca9d86445",
     "timeStamp": 1758822431
   }
   ```
6. **testErrRes** Test Endpoint
   - HTTP Method: GET
   - Request Path: /api/v1/test/testErrRes
   - Request Parameters: None
   - Response Example:
   ```json
   {
     "data": {
       "code": 10001,
       "message": "Server exception"
     },
     "requestId": "79768196-75cc-4b9e-8286-998a4bd4218b",
     "timeStamp": 1758822435
   }
   ```
7. **testErrorMiddleware** Test Endpoint

   - HTTP Method: GET
   - Request Path: /api/v1/test/testErrorMiddleware
   - Request Parameters: None
   - Response Example: Recovery middleware handles panic and returns empty response

8. **testLongReq** Test Endpoint
   - HTTP Method: POST
   - Request Path: /api/v2/test/testLongReq
   - Request Parameters:
   ```json
   {
     "duration": 3
   }
   ```
   - Response Example:
   ```json
   {
     "data": {
       "duration": 3,
       "message": "Simulated long-running request completed!"
     },
     "requestId": "caecc92a-0e04-4b4a-ac9e-cdbba2cc34ad",
     "timeStamp": 1758822445
   }
   ```

## torrent Module

1. **parseMagnet** Parse Magnet URI
   - HTTP Method: POST
   - Request Path: /api/v1/torrent/parse
   - Request Parameters:
   ```json
   {
     "magnet_uri": "magnet:?xt=urn:btih:...",
     "trackers": ["udp://tracker.example.com:1337/announce"]
   }
   ```
   - Response Example:
   ```json
   {
     "data": {
       "info_hash": "abc123...",
       "name": "Example Torrent",
       "total_size": 1073741824,
       "files": [
         {
           "index": 0,
           "path": "video.mp4",
           "size": 1073741824,
           "size_readable": "1.0 GB",
           "is_streamable": true
         }
       ]
     },
     "requestId": "...",
     "timeStamp": 1758822417
   }
   ```

2. **startDownload** Start Download
   - HTTP Method: POST
   - Request Path: /api/v1/torrent/download
   - Request Parameters:
   ```json
   {
     "info_hash": "abc123...",
     "selected_files": [0, 1, 2],
     "trackers": []
   }
   ```
   - Response Example:
   ```json
   {
     "data": {
       "info_hash": "abc123...",
       "message": "Download started successfully"
     },
     "requestId": "...",
     "timeStamp": 1758822417
   }
   ```

3. **getProgress** Get Download Progress
   - HTTP Method: GET
   - Request Path: /api/v1/torrent/progress/:info_hash
   - Request Parameters: info_hash in path
   - Response Example:
   ```json
   {
     "data": {
       "info_hash": "abc123...",
       "name": "Example Torrent",
       "total_size": 1073741824,
       "downloaded_size": 536870912,
       "progress": 50.0,
       "status": "downloading",
       "peers": 10,
       "seeds": 5
     },
     "requestId": "...",
     "timeStamp": 1758822417
   }
   ```

4. **pauseDownload** Pause Download
   - HTTP Method: POST
   - Request Path: /api/v1/torrent/pause
   - Request Parameters:
   ```json
   {
     "info_hash": "abc123..."
   }
   ```
   - Response Example:
   ```json
   {
     "data": {
       "info_hash": "abc123...",
       "message": "Download paused successfully"
     },
     "requestId": "...",
     "timeStamp": 1758822417
   }
   ```

5. **resumeDownload** Resume Download
   - HTTP Method: POST
   - Request Path: /api/v1/torrent/resume
   - Request Parameters:
   ```json
   {
     "info_hash": "abc123...",
     "selected_files": [0, 1, 2]
   }
   ```
   - Response Example:
   ```json
   {
     "data": {
       "info_hash": "abc123...",
       "message": "Download resumed successfully"
     },
     "requestId": "...",
     "timeStamp": 1758822417
   }
   ```

6. **removeTorrent** Remove Torrent
   - HTTP Method: POST
   - Request Path: /api/v1/torrent/remove
   - Request Parameters:
   ```json
   {
     "info_hash": "abc123...",
     "delete_files": true
   }
   ```
   - Response Example:
   ```json
   {
     "data": {
       "info_hash": "abc123...",
       "message": "Torrent removed successfully"
     },
     "requestId": "...",
     "timeStamp": 1758822417
   }
   ```

7. **listTorrents** List All Torrents
   - HTTP Method: GET
   - Request Path: /api/v1/torrent/list
   - Request Parameters: None
   - Response Example:
   ```json
   {
     "data": {
       "torrents": [
         {
           "info_hash": "abc123...",
           "name": "Example Torrent",
           "total_size": 1073741824,
           "progress": 50.0,
           "status": 1,
           "poster_path": "/path/to/poster.jpg",
           "created_at": 1758822417
         }
       ],
       "total": 1
     },
     "requestId": "...",
     "timeStamp": 1758822417
   }
   ```

8. **getTorrentDetail** Get Torrent Details
   - HTTP Method: GET
   - Request Path: /api/v1/torrent/detail/:info_hash
   - Request Parameters: info_hash in path
   - Response Example:
   ```json
   {
     "data": {
       "info_hash": "abc123...",
       "name": "Example Torrent",
       "total_size": 1073741824,
       "files": [
         {
           "index": 0,
           "path": "video.mp4",
           "size": 1073741824,
           "size_readable": "1.0 GB",
           "is_streamable": true
         }
       ],
       "poster_path": "/path/to/poster.jpg",
       "download_path": "./download/abc123...",
       "status": 1,
       "progress": 50.0,
       "created_at": 1758822417
     },
     "requestId": "...",
     "timeStamp": 1758822417
   }
   ```

9. **serveFile** Serve Downloaded File (Streaming Support)
   - HTTP Method: GET
   - Request Path: /api/v1/torrent/file/:info_hash/*file_path
   - Request Parameters: info_hash and file_path in path
   - Supports HTTP Range requests for video streaming
   - Response: Binary file content with appropriate Content-Type header

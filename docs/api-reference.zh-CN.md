# 接口文档

## 统一响应格式：

- 正确响应：

```json
{
  "data": any,
  "requestId": string,
  "timeStamp": number
}
```

- 错误响应：

```json
{
  "code": number,
  "msg": string,
  "data": any,
  "requestId": string,
  "timeStamp": number
}
```

## test 测试模块

1. **testPing** 测试接口
   - 请求方式：GET
   - 请求路径：/api/v1/test/testPing
   - 请求参数：无
   - 响应示例：
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
2. **testHello** 测试接口
   - 请求方式：GET
   - 请求路径：/api/v1/test/testHello
   - 请求参数：无
   - 响应示例：
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
3. **testLogger** 测试接口
   - 请求方式：GET
   - 请求路径：/api/v1/test/testLogger
   - 请求参数：无
   - 响应示例：
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
4. **testRedis** 测试接口
   - 请求方式：POST
   - 请求路径：/api/v1/test/testRedis
   - 请求参数：
   ```json
   {
     "key": "test",
     "value": "hello",
     "ttl": 60
   }
   ```
   - 响应示例：
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
5. **testSuccessRes** 测试接口
   - 请求方式：GET
   - 请求路径：/api/v1/test/testSuccessRes
   - 请求参数：无
   - 响应示例：
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
6. **testErrRes** 测试接口
   - 请求方式：GET
   - 请求路径：/api/v1/test/testErrRes
   - 请求参数：无
   - 响应示例：
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
7. **testErrorMiddleware** 测试接口

   - 请求方式：GET
   - 请求路径：/api/v1/test/testErrorMiddleware
   - 请求参数：无
   - 响应示例：Recovery 中间件处理 panic 并返回空响应

8. **testLongReq** 测试接口
   - 请求方式：POST
   - 请求路径：/api/v2/test/testLongReq
   - 请求参数：
   ```json
   {
     "duration": 3
   }
   ```
   - 响应示例：
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

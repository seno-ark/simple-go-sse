# simple-go-sse
Simple SSE (Server Sent Event) implementation using Golang.

### Run the project
```
git clone git@github.com:seno-ark/simple-go-sse.git
cd simple-go-sse
go run .
```

### Send message
```
curl "localhost:3000/send" -i -d "username=Foo" -d "message=Hello, world!"
```

### Listen to server events
#### In terminal:
```
curl "localhost:3000/sse?username=Bar"
```
#### In Browser:
```
open http://localhost:3000
```

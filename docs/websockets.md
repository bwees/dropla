# Adding Socket.IO WebSockets

The template ships with a plain HTTP API. When you need the server to push
updates to the browser, add a [Socket.IO](https://socket.io) channel alongside
the REST controllers. Socket.IO messages are **not** described by OpenAPI, so
this lives next to the generated SDK rather than inside it.

## Server

1. Add the dependencies:

   ```sh
   cd server
   go get github.com/zishang520/socket.io/v2/socket github.com/zishang520/engine.io/v2/types
   ```

2. Create `internal/socketio/socketio.go` that wraps a `socket.Server` and
   exposes an `Emit(event string, payload any)` method plus an `http.Handler`:

   ```go
   package socketio

   import (
       "encoding/json"
       "net/http"

       "github.com/zishang520/socket.io/v2/socket"
       "go.uber.org/fx"
   )

   const path = "/api/ws"

   type SocketIO struct {
       io *socket.Server
   }

   func NewSocketIO() *SocketIO {
       opts := socket.DefaultServerOptions()
       opts.SetPath(path)
       return &SocketIO{io: socket.NewServer(nil, opts)}
   }

   func (s *SocketIO) Emit(event string, payload any) {
       data, err := json.Marshal(payload)
       if err != nil {
           return
       }
       s.io.Emit(event, string(data))
   }

   func (s *SocketIO) Handler() http.Handler {
       return s.io.ServeHandler(nil)
   }

   var Module = fx.Provide(NewSocketIO)
   ```

3. Register the handler from a controller. Socket.IO speaks over a `/ws/`
   route, so mount it with `fuego.Handle` (not `fuego.Get`) and hide it from
   the spec:

   ```go
   func (c *WebsocketController) Register(api *fuego.Server) {
       fuego.Handle(api, "/ws/", c.socket.Handler())
   }
   ```

4. Wire `socketio.Module` into `fx.New(...)` in `main.go` and add the new
   controller to `controllers.Module`.

Inject `*socketio.SocketIO` into any service and call `Emit` to broadcast.

## Client

1. Install the client:

   ```sh
   cd web
   pnpm add socket.io-client
   ```

2. Connect from a Svelte module. The Vite dev proxy already forwards
   `/api` with `ws: true`, so the same relative URL works in dev and prod:

   ```ts
   import { io } from 'socket.io-client';

   const socket = io({ path: '/api/ws' });

   socket.on('note:created', (raw: string) => {
       const note = JSON.parse(raw);
       // update your $state here
   });
   ```

## Origin checks

Browsers send an `Origin` header on the WebSocket upgrade. In production the
frontend and API share an origin, but in dev the frontend runs on Vite
(`http://localhost:5173`). If you enable an allow-list, read it from an env var
in `internal/config` and pass it to `opts.SetAllowRequest(...)`.

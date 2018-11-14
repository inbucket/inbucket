// Register the websocket listeners for the monitor API.
export default function registerMonitorPorts(app) {
  var uri = '/api/v1/monitor/messages'
  var url = ((window.location.protocol === "https:") ? "wss://" : "ws://") + window.location.host + uri

  // Current handler.
  var handler = null

  app.ports.monitorCommand.subscribe(function (cmd) {
    if (handler != null) {
      handler.down()
      handler = null
    }
    if (cmd) {
      // Command is up.
      handler = websocketHandler(url, app.ports.monitorMessage)
      handler.up()
    }
  })
}

// Creates a handler responsible for connecting, disconnecting from web socket.
function websocketHandler(url, port) {
  var ws = null

  return {
    up: () => {
      ws = new WebSocket(url)

      ws.addEventListener('open', function (e) {
        port.send(true)
      })
      ws.addEventListener('close', function (e) {
        port.send(false)
      })
      ws.addEventListener('message', function (e) {
        var msg = JSON.parse(e.data)
        port.send(msg)
      })
    },

    down: () => {
      ws.close()
    }
  }
}

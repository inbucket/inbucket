import './main.css'
import { Elm } from './Main.elm'
import registerMonitorPorts from './registerMonitor'
import './renderedHtml'

// App startup.
var app = Elm.Main.init({
  node: document.getElementById('root'),
  flags: sessionObject()
})

// Message monitor.
registerMonitorPorts(app)

// Session storage.
app.ports.storeSession.subscribe(function (session) {
  localStorage.session = JSON.stringify(session)
})

window.addEventListener("storage", function (event) {
  if (event.storageArea === localStorage && event.key === "session") {
    app.ports.onSessionChange.send(sessionObject())
  }
}, false)

function sessionObject() {
  var s = localStorage.session
  try {
    if (s) {
      return JSON.parse(s)
    }
  } catch (error) {
    console.error(error)
  }
  return null
}

// Window title.
app.ports.windowTitle.subscribe(function (title) {
  document.title = title
})

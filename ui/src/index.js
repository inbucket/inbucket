import './main.css'
import { Main } from './Main.elm'
import registerServiceWorker from './registerServiceWorker'
import registerMonitorPorts from './registerMonitor'

// App startup.
var app = Main.embed(document.getElementById('root'), sessionObject())

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

registerServiceWorker()

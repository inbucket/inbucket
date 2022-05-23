import '@fortawesome/fontawesome-free/css/all.css'
import '@webcomponents/webcomponentsjs/webcomponents-bundle'
import 'opensans-npm-webfont'
import { Elm } from './Main.elm'
import './monitorMessages'
import './renderedHtml'

// Initial configuration from Inbucket server to Elm App.
var flags = {
  "app-config": appConfig(),
  "session": sessionObject(),
}

// App startup.
var app = Elm.Main.init({
  node: document.getElementById('root'),
  flags: flags,
})

// Session storage.
app.ports.storeSession.subscribe(function (session) {
  localStorage.session = JSON.stringify(session)
})

window.addEventListener("storage", function (event) {
  if (event.storageArea === localStorage && event.key === "session") {
    app.ports.onSessionChange.send(sessionObject())
  }
}, false)

// Decode the JSON value of the app-config cookie, then delete it.
function appConfig() {
  var name = "app-config"
  var c = getCookie(name)
  if (c) {
    deleteCookie(name)
    return JSON.parse(decodeURIComponent(c))
  }
  console.warn("Inbucket " + name + " cookie not found, running with defaults.")
  return {
    "monitor-visible": true,
  }
}

// Grab peristent session data out of local storage.
function sessionObject() {
  try {
    var s = localStorage.session
    if (s) {
      return JSON.parse(s)
    }
  } catch (error) {
    console.error(error)
  }
  return null
}

function getCookie(cookieName) {
  var name = cookieName + "="
  var cookies = decodeURIComponent(document.cookie).split(';')
  for (var i=0; i<cookies.length; i++) {
    var cookie = cookies[i].trim()
    if (cookie.indexOf(name) == 0) {
      return cookie.substring(name.length, cookie.length)
    }
  }
  return null
}

function deleteCookie(cookieName) {
  document.cookie = cookieName +
    "=; expires=Thu, 01 Jan 1970 00:00:00 UTC; path=/;"
}

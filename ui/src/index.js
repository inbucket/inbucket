import './main.css';
import { Main } from './Main.elm';
import registerServiceWorker from './registerServiceWorker';

var app = Main.embed(document.getElementById('root'), sessionObject());

app.ports.storeSession.subscribe(function (session) {
  localStorage.session = JSON.stringify(session);
});

app.ports.windowTitle.subscribe(function (title) {
  document.title = title;
});

window.addEventListener("storage", function (event) {
  if (event.storageArea === localStorage && event.key === "session") {
    app.ports.onSessionChange.send(sessionObject());
  }
}, false);

function sessionObject() {
  var s = localStorage.session;
  try {
    if (s) {
      return JSON.parse(s);
    }
  } catch (error) {
    console.error(error);
  }
  return null;
}

registerServiceWorker();

// monitor-messages connects to the Inbucket backend via WebSocket to monitor
// incoming messages.
customElements.define(
  'monitor-messages',
  class MonitorMessages extends HTMLElement {
    constructor() {
      const self = super()
      // TODO make URI/URL configurable.
      var uri = '/api/v1/monitor/messages'
      self._url = ((window.location.protocol === 'https:') ? 'wss://' : 'ws://') + window.location.host + uri
      self._socket = null
    }

    connectedCallback() {
      const self = this
      self._socket = new WebSocket(self._url)
      var ws = self._socket
      ws.addEventListener('open', function (e) {
        self.dispatchEvent(new CustomEvent('connected', { detail: true }))
      })
      ws.addEventListener('close', function (e) {
        self.dispatchEvent(new CustomEvent('connected', { detail: false }))
      })
      ws.addEventListener('message', function (e) {
        self.dispatchEvent(new CustomEvent('message', {
          detail: JSON.parse(e.data),
        }))
      })
    }

    disconnectedCallback() {
      var ws = this._socket
      if (ws) {
        ws.close()
      }
    }
  }
)

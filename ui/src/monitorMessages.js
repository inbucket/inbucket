// monitor-messages connects to the Inbucket backend via WebSocket to monitor
// incoming messages.
customElements.define(
  'monitor-messages',
  class MonitorMessages extends HTMLElement {
    static get observedAttributes() {
      return [ 'src' ]
    }

    constructor() {
      super()
      this._url = null    // Current websocket URL.
      this._socket = null // Currently open WebSocket.
    }

    connectedCallback() {
      if (this.hasAttribute('src')) {
        this.wsOpen(this.getAttribute('src'))
      }
    }

    attributeChangedCallback() {
      // Checking _socket prevents connection attempts prior to connectedCallback().
      if (this._socket && this.hasAttribute('src')) {
        this.wsOpen(this.getAttribute('src'))
      }
    }

    disconnectedCallback() {
      this.wsClose()
    }

    // Connects to WebSocket and registers event listeners.
    wsOpen(uri) {
      const url =
        ((window.location.protocol === 'https:') ? 'wss://' : 'ws://') +
        window.location.host + uri
      if (this._socket && url === this._url) {
        // Already connected to same URL.
        return
      }
      this.wsClose()
      this._url = url

      console.info("Connecting to WebSocket", url)
      const ws = new WebSocket(url)
      this._socket = ws

      // Register event listeners.
      const self = this
      ws.addEventListener('open', function (_e) {
        self.dispatchEvent(new CustomEvent('connected', { detail: true }))
      })
      ws.addEventListener('close', function (_e) {
        self.dispatchEvent(new CustomEvent('connected', { detail: false }))
      })
      ws.addEventListener('message', function (e) {
        self.dispatchEvent(new CustomEvent('message', {
          detail: JSON.parse(e.data),
        }))
      })
    }

    // Closes WebSocket connection.
    wsClose() {
      const ws = this._socket
      if (ws) {
        ws.close()
      }
    }

    get src() {
      return this.getAttribute('src')
    }

    set src(value) {
      this.setAttribute('src', value)
    }
  }
)

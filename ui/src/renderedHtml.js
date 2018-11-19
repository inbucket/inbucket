// This element allows Inbucket to draw server rendered HTML, aka HTML email.
// https://leveljournal.com/server-rendered-html-in-elm

customElements.define(
  "rendered-html",
  class RenderedHtml extends HTMLElement {
    constructor() {
      super()
      this._content = ""
    }

    set content(value) {
      if (this._content === value) {
        return
      }
      this._content = value
      this.innerHTML = value
    }

    get content() {
      return this._content
    }
  }
)

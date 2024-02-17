package web

import (
	"html/template"
	"net/http"
	"os"

	"github.com/rs/zerolog/log"
)

// Handler is a function type that handles an HTTP request in Inbucket.
type Handler func(http.ResponseWriter, *http.Request, *Context) error

// ServeHTTP builds the context and passes onto the real handler.
func (h Handler) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	// Create the context.
	ctx, err := NewContext(req)
	if err != nil {
		log.Error().Str("module", "web").Err(err).Msg("HTTP failed to create context")
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer ctx.Close()

	// Run the handler, grab the error, and report it.
	err = h(w, req, ctx)
	if err != nil {
		log.Error().Str("module", "web").Str("path", req.RequestURI).Err(err).
			Msg("Error handling request")
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

// cookieHandler injects an HTTP cookie into the response.
func cookieHandler(cookie *http.Cookie, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		log.Debug().Str("module", "web").Str("remote", req.RemoteAddr).Str("proto", req.Proto).
			Str("method", req.Method).Str("path", req.RequestURI).Msg("Injecting cookie")
		http.SetCookie(w, cookie)
		next.ServeHTTP(w, req)
	})
}

// fileHandler creates a handler that sends the named file regardless of the requested URL.
func fileHandler(name string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		f, err := os.Open(name)
		if err != nil {
			log.Error().Str("module", "web").Str("path", req.RequestURI).Str("file", name).Err(err).
				Msg("Error opening file")
			http.Error(w, "Error opening file", http.StatusInternalServerError)
			return
		}
		defer f.Close()

		d, err := f.Stat()
		if err != nil {
			log.Error().Str("module", "web").Str("path", req.RequestURI).Str("file", name).Err(err).
				Msg("Error stating file")
			http.Error(w, "Error opening file", http.StatusInternalServerError)
			return
		}
		http.ServeContent(w, req, d.Name(), d.ModTime(), f)
	})
}

// noMatchHandler creates a handler to log requests that Gorilla mux is unable to route,
// returning specified statusCode to the client.
func noMatchHandler(statusCode int, message string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		log.Warn().Str("module", "web").Str("remote", req.RemoteAddr).Str("proto", req.Proto).
			Str("method", req.Method).Str("path", req.RequestURI).Msg(message)
		w.WriteHeader(statusCode)
	})
}

// requestLoggingWrapper returns middleware that logs client requests.
func requestLoggingWrapper(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		log.Debug().Str("module", "web").Str("remote", req.RemoteAddr).Str("proto", req.Proto).
			Str("method", req.Method).Str("path", req.RequestURI).Msg("Request")
		next.ServeHTTP(w, req)
	})
}

// spaTemplateHandler creates a handler to serve the index.html template for our SPA.
func spaTemplateHandler(tmpl *template.Template, basePath string) http.Handler {
	tmplData := struct {
		BasePath string
	}{
		BasePath: basePath,
	}

	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		// ensure we do now allow click jacking
		w.Header().Set("X-Frame-Options", "SameOrigin")
		err := tmpl.Execute(w, tmplData)
		if err != nil {
			log.Error().Str("module", "web").Str("remote", req.RemoteAddr).Str("proto", req.Proto).
				Str("method", req.Method).Str("path", req.RequestURI).Err(err).
				Msg("Error rendering SPA index template")
		}
	})
}

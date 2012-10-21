package web

import (
	"github.com/jhillyerd/inbucket"
	"html/template"
	"path/filepath"
	"sync"
)

var cachedTemplates = map[string]*template.Template{}
var cachedMutex sync.Mutex

func T(name string) *template.Template {
	cachedMutex.Lock()
	defer cachedMutex.Unlock()

	if t, ok := cachedTemplates[name]; ok {
		return t
	}

	templateDir := inbucket.GetWebConfig().TemplateDir
	templateFile := filepath.Join(templateDir, name)
	inbucket.Trace("Parsing template %v", templateFile)

	t := template.New("_base.html").Funcs(TemplateFuncs)
	t = template.Must(t.ParseFiles(
		filepath.Join(templateDir, "_base.html"),
		templateFile,
	))
	cachedTemplates[name] = t

	return t
}

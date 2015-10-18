package gohaml

import (
	"io"
	"strings"

	"github.com/flosch/pongo2"
	"github.com/revel/revel"
)

// Adapter for HAML Templates.
type PongoTemplate struct {
	name     string
	template *pongo2.Template
	engine   *PongoEngine
}

func (tmpl PongoTemplate) Name() string {
	return tmpl.name
}

// return a 'revel.Template' from HAML's template.
func (tmpl PongoTemplate) Render(wr io.Writer, arg interface{}) (err error) {
	return tmpl.template.ExecuteWriter(pongo2.Context(arg.(map[string]interface{})), wr)
}

func (tmpl PongoTemplate) Content() []string {
	content, _ := revel.ReadLines(tmpl.engine.loader.TemplatePaths[tmpl.Name()])
	return content
}

type PongoEngine struct {
	loader                *revel.TemplateLoader
	templateSetBybasePath map[string]*pongo2.TemplateSet
	templates             map[string]*pongo2.Template
}

func (engine *PongoEngine) ParseAndAdd(templateName string, templateSource string, basePath string) *revel.Error {
	templateSet := engine.templateSetBybasePath[basePath]
	if nil == templateSet {
		templateSet = pongo2.NewSet(basePath, pongo2.MustNewLocalFileSystemLoader(basePath))
		engine.templateSetBybasePath[basePath] = templateSet
	}

	tpl, err := templateSet.FromString(templateSource)
	if nil != err {
		_, line, description := parseHamlError(err)
		return &revel.Error{
			Title:       "Template Compilation Error",
			Path:        templateName,
			Description: description,
			Line:        line,
			SourceLines: strings.Split(templateSource, "\n"),
		}
	}

	engine.templates[strings.ToLower(templateName)] = tpl
	return nil
}

func parseHamlError(err error) (templateName string, line int, description string) {
	pongoError := err.(*pongo2.Error)
	if nil != pongoError {
		return pongoError.Filename, pongoError.Line, pongoError.ErrorMsg
	}
	return "", 0, err.Error()
}

func (engine *PongoEngine) Lookup(templateName string) revel.Template {
	tpl := engine.templates[strings.ToLower(templateName)]
	if nil == tpl {
		return nil
	}
	return PongoTemplate{templateName, tpl, engine}
}

func init() {
	revel.TemplateEngines[revel.GOHAML_TEMPLATE] = func(loader *revel.TemplateLoader) (revel.TemplateEngine, error) {
		return &PongoEngine{
			loader:                loader,
			templateSetBybasePath: map[string]*pongo2.TemplateSet{},
			templates:             map[string]*pongo2.Template{},
		}, nil
	}
}

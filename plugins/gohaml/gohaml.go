package gohaml

import (
	"io"
	"regexp"
	"strconv"
	"strings"

	"github.com/realistschuckle/gohaml"
	"github.com/revel/revel"
)

// Adapter for HAML Templates.
type HamlTemplate struct {
	name     string
	template *gohaml.Engine
	engine   *HamlEngine
}

func (haml HamlTemplate) Name() string {
	return haml.name
}

// return a 'revel.Template' from HAML's template.
func (haml HamlTemplate) Render(wr io.Writer, arg interface{}) (err error) {
	_, err = io.WriteString(wr, haml.template.Render(arg.(map[string]interface{})))
	return
}

func (haml HamlTemplate) Content() []string {
	content, _ := revel.ReadLines(haml.engine.loader.TemplatePaths[haml.Name()])
	return content
}

type HamlEngine struct {
	loader      *revel.TemplateLoader
	templateSet map[string]*gohaml.Engine
}

func (engine *HamlEngine) ParseAndAdd(templateName string, templateSource string, basePath string) *revel.Error {
	tpl, err := gohaml.NewEngine(templateSource)
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

	engine.templateSet[strings.ToLower(templateName)] = tpl
	return nil
}

func parseHamlError(err error) (templateName string, line int, description string) {
	description = err.Error()
	i := regexp.MustCompile(`line\s*\d+:`).FindStringIndex(description)
	if i != nil {
		line, err = strconv.Atoi(strings.TrimSpace(description[i[0]+4 : i[1]-1]))
		if err != nil {
			revel.ERROR.Println("Failed to parse line number from error message:", err)
		}
		description = strings.TrimSpace(description[i[1]+1:])
	}
	return templateName, line, description
}

func (engine *HamlEngine) Lookup(templateName string) revel.Template {
	tpl := engine.templateSet[strings.ToLower(templateName)]
	if nil == tpl {
		return nil
	}
	return HamlTemplate{templateName, tpl, engine}
}

func init() {
	revel.TemplateEngines[revel.GOHAML_TEMPLATE] = func(loader *revel.TemplateLoader) (revel.TemplateEngine, error) {
		return &HamlEngine{
			loader:      loader,
			templateSet: map[string]*gohaml.Engine{},
		}, nil
	}
}

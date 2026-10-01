package mail

import (
	"embed"
	"fmt"
	"html/template"
	"path/filepath"
	"strings"
)

//go:embed templates/*.html
var embeddedTemplates embed.FS

func parseTemplate(templatePath string, templateName string) (*template.Template, error) {
	name := filepath.Base(strings.TrimSpace(templateName))
	if name == "." || name == "" {
		return nil, fmt.Errorf("template name is required")
	}
	if strings.TrimSpace(templatePath) != "" {
		return template.ParseFiles(filepath.Join(templatePath, name))
	}
	return template.ParseFS(embeddedTemplates, "templates/"+name)
}

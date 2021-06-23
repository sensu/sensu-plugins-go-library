package templates

import (
	"bytes"
	"fmt"
	"github.com/Masterminds/sprig"
	"github.com/google/uuid"
	"os"
	"text/template"
	"time"
)

func EvalTemplate(templName, templStr string, templSrc interface{}) (string, error) {
	if templSrc == nil {
		return "", fmt.Errorf("must pass in template source")
	}
	if len(templStr) == 0 {
		return "", fmt.Errorf("must pass in template")
	}

	templ, err := template.New(templName).Funcs(sprig.TxtFuncMap()).Funcs(template.FuncMap{
		"UnixTime":      func(i int64) time.Time { return time.Unix(i, 0) },
		"UUIDFromBytes": uuid.FromBytes,
		"Hostname":      os.Hostname,
	}).Parse(templStr)
	if err != nil {
		return "", fmt.Errorf("Error building template: %s", err)
	}

	buf := new(bytes.Buffer)
	err = templ.Execute(buf, templSrc)
	if err != nil {
		return "", fmt.Errorf("Error executing template: %s", err)
	}

	return buf.String(), nil
}

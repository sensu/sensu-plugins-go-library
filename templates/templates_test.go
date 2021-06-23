package templates

import (
	"encoding/json"
	"github.com/google/uuid"
	"github.com/sensu/sensu-go/types"
	"github.com/stretchr/testify/assert"
	"os"
	"testing"
	"time"
)

var (
	templateOk          = "Check: {{ .Check.Name }} Entity: {{ .Entity.Name }} !"
	templateOkUnixTime  = "Check: {{ .Check.Name }} Entity: {{ .Entity.Name }} Executed: {{(UnixTime .Check.Executed).Format \"2 Jan 2006 15:04:05\"}} !"
	templateOkUUID      = "Check: {{ .Check.Name }} Entity: {{ .Entity.Name }} Event ID: {{UUIDFromBytes .ID}} !"
	templateOkHostname  = "Check: {{ .Check.Name }} Entity: {{ .Entity.Name }} Hostname: {{Hostname}} !"
	templateVarNotFound = "Check: {{ .Check.NameZZZ }} Entity: {{ .Entity.Name }} !"
	templateInvalid     = "Check: {{ .Check.Name Entity: {{ .Entity.Name }} !"
	templateReplace     = "Check: {{ .Check.Name | replace \"nginx\" \"sprig\" }}"
)

// Valid test
func TestEvalTemplate_Valid(t *testing.T) {
	event := &types.Event{}
	_ = json.Unmarshal(testEventBytes, event)

	result, err := EvalTemplate("templOk", templateOk, event)
	assert.Nil(t, err)
	assert.Equal(t, "Check: check-nginx Entity: webserver01 !", result)
}

// Valid test - Time Check
func TestEvalTemplateUnixTime_Valid(t *testing.T) {
	event := &types.Event{}
	_ = json.Unmarshal(testEventBytes, event)

	executed := time.Unix(event.Check.Executed, 0).Format("2 Jan 2006 15:04:05")
	result, err := EvalTemplate("templOk", templateOkUnixTime, event)
	assert.Nil(t, err)
	assert.Equal(t, "Check: check-nginx Entity: webserver01 Executed: "+executed+" !", result)
}

// Valid test - UUID
func TestEvalTemplateUUIDValid(t *testing.T) {
	event := &types.Event{}
	_ = json.Unmarshal(testEventBytes, event)

	uuidFromEvent, _ := uuid.FromBytes(event.ID)
	result, err := EvalTemplate("templOk", templateOkUUID, event)
	assert.Nil(t, err)
	assert.Equal(t, "Check: check-nginx Entity: webserver01 Event ID: "+uuidFromEvent.String()+" !", result)
}

// Valid test - Hostname
func TestEvalTemplateHostname(t *testing.T) {
	event := &types.Event{}
	_ = json.Unmarshal(testEventBytes, event)

	hostname, _ := os.Hostname()
	result, err := EvalTemplate("templOk", templateOkHostname, event)
	assert.Nil(t, err)
	assert.Equal(t, "Check: check-nginx Entity: webserver01 Hostname: "+hostname+" !", result)
}

// Variable not found
func TestEvalTemplate_VarNotFound(t *testing.T) {
	event := &types.Event{}
	_ = json.Unmarshal(testEventBytes, event)

	result, err := EvalTemplate("templOk", templateVarNotFound, event)
	assert.Equal(t, "", result)
	assert.NotNil(t, err)
}

// Nil template source
func TestEvalTemplate_NilSource(t *testing.T) {
	result, err := EvalTemplate("templOk", templateVarNotFound, nil)
	assert.Equal(t, "", result)
	assert.NotNil(t, err)
}

// Empty template
func TestEvalTemplate_NilTemplate(t *testing.T) {
	event := &types.Event{}
	_ = json.Unmarshal(testEventBytes, event)

	result, err := EvalTemplate("templOk", "", event)
	assert.Equal(t, "", result)
	assert.NotNil(t, err)
}

// Invalid template
func TestEvalTemplate_InvalidTemplate(t *testing.T) {
	event := &types.Event{}
	_ = json.Unmarshal(testEventBytes, event)

	result, err := EvalTemplate("templOk", templateInvalid, event)
	assert.Equal(t, "", result)
	assert.NotNil(t, err)
}

// Template making use of sprig function replace
func TestEvalTemplate_SprigReplace(t *testing.T) {
	event := &types.Event{}
	_ = json.Unmarshal(testEventBytes, event)

	result, err := EvalTemplate("templSprigReplace", templateReplace, event)
	assert.Nil(t, err)
	assert.Equal(t, "Check: check-sprig", result)
}

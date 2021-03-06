package sensu

import (
	"fmt"
	"log"
	"os"

	"github.com/sensu/sensu-go/types"
)

const (
	CheckStateOK       = 0
	CheckStateWarning  = 1
	CheckStateCritical = 2
	CheckStateUnknown  = 3
)

type GoCheck struct {
	basePlugin
	validationFunction func(event *types.Event) (int, error)
	executeFunction    func(event *types.Event) (int, error)
}

func NewGoCheck(config *PluginConfig, options []*PluginConfigOption,
	validationFunction func(*types.Event) (int, error),
	executeFunction func(*types.Event) (int, error), readEvent bool) *GoCheck {
	check := &GoCheck{
		basePlugin: basePlugin{
			config:                 config,
			options:                options,
			sensuEvent:             nil,
			eventReader:            os.Stdin,
			eventValidation:        false,
			readEvent:              readEvent,
			configurationOverrides: true,
			errorExitStatus:        1,
		},
		validationFunction: validationFunction,
		executeFunction:    executeFunction,
	}

	check.pluginWorkflowFunction = check.goCheckWorkflow
	if err := check.initPlugin(); err != nil {
		log.Printf("failed to initialize check plugin: %s", err)
	}

	return check
}

// Executes the check
func (goCheck *GoCheck) goCheckWorkflow(_ []string) (int, error) {
	// Validate input using validateFunction
	status, err := goCheck.validationFunction(goCheck.sensuEvent)
	if err != nil {
		return status, fmt.Errorf("error validating input: %s", err)
	}

	// Execute check logic using executeFunction
	status, err = goCheck.executeFunction(goCheck.sensuEvent)
	if err != nil {
		return status, fmt.Errorf("error executing check: %s", err)
	}

	return status, nil
}

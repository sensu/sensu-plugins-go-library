package sensu

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"testing"

	"github.com/sensu/sensu-go/types"
	"github.com/stretchr/testify/assert"
)

func init() {
	log.SetOutput(ioutil.Discard)
}

type handlerValues struct {
	arg1 string
	arg2 uint64
	arg3 bool
	arg4 []string
	arg5 []string
}

var (
	defaultHandlerConfig = PluginConfig{
		Name:     "TestHandler",
		Short:    "Short Description",
		Timeout:  10,
		Keyspace: "sensu.io/plugins/segp/config",
	}

	defaultOption1 = PluginConfigOption{
		Argument:  "arg1",
		Default:   "Default1",
		Env:       "ENV_1",
		Path:      "path1",
		Shorthand: "d",
		Usage:     "First argument",
		Secret:    true,
	}

	defaultOption2 = PluginConfigOption{
		Argument:  "arg2",
		Default:   uint64(33333),
		Env:       "ENV_2",
		Path:      "path2",
		Shorthand: "e",
		Usage:     "Second argument",
	}

	defaultOption3 = PluginConfigOption{
		Argument:  "arg3",
		Default:   false,
		Env:       "ENV_3",
		Path:      "path3",
		Shorthand: "f",
		Usage:     "Third argument",
	}

	defaultOption4 = PluginConfigOption{
		Argument:  "arg4",
		Default:   []string{},
		Env:       "ENV_4",
		Path:      "path4",
		Shorthand: "g",
		Usage:     "Fourth argument",
	}

	defaultOption5 = PluginConfigOption{
		Argument:  "arg5",
		Default:   []string{},
		Env:       "ENV_5",
		Path:      "path5",
		Shorthand: "i",
		Usage:     "Fifth argument",
		Array:     true,
	}

	defaultCmdLineArgs = []string{"--arg1", "value-arg1", "--arg2", "7531", "--arg3=false",
		"--arg4=hey,you",
		"--arg4=clap,hands",
		`--arg5="this,and,that"`, `--arg5="now,wat"`}
)

func TestNewGoHandler(t *testing.T) {
	values := &handlerValues{}
	options := getHandlerOptions(values, false)
	goHandler := NewGoHandler(&defaultHandlerConfig, options, func(event *types.Event) error {
		return nil
	}, func(event *types.Event) error {
		return nil
	})

	assert.NotNil(t, goHandler)
	assert.NotNil(t, goHandler.options)
	assert.Equal(t, options, goHandler.options)
	assert.NotNil(t, goHandler.config)
	assert.Equal(t, &defaultHandlerConfig, goHandler.config)
	assert.NotNil(t, goHandler.validationFunction)
	assert.NotNil(t, goHandler.executeFunction)
	assert.Nil(t, goHandler.sensuEvent)
	assert.Equal(t, os.Stdin, goHandler.eventReader)
}
func TestNewGoHandlerWithNilOptionDefaults(t *testing.T) {
	values := &handlerValues{}
	options := getHandlerOptions(values, true)
	goHandler := NewGoHandler(&defaultHandlerConfig, options, func(event *types.Event) error {
		return nil
	}, func(event *types.Event) error {
		return nil
	})

	assert.NotNil(t, goHandler)
	assert.NotNil(t, goHandler.options)
	assert.Equal(t, options, goHandler.options)
	assert.NotNil(t, goHandler.config)
	assert.Equal(t, &defaultHandlerConfig, goHandler.config)
	assert.NotNil(t, goHandler.validationFunction)
	assert.NotNil(t, goHandler.executeFunction)
	assert.Nil(t, goHandler.sensuEvent)
	assert.Equal(t, os.Stdin, goHandler.eventReader)
}

func TestNewGoHandler_NoOptionValue(t *testing.T) {
	var exitStatus int
	options := getHandlerOptions(nil, false)
	handlerConfig := defaultHandlerConfig

	goHandler := NewGoHandler(&handlerConfig, options,
		func(event *types.Event) error {
			return nil
		}, func(event *types.Event) error {
			return nil
		})

	goHandler.exitFunction = func(i int) {
		exitStatus = i
	}
	goHandler.Execute()
	assert.Equal(t, 1, exitStatus)
}
func TestNewGoHandler_NoOptionValueWithNilDefaults(t *testing.T) {
	var exitStatus int
	options := getHandlerOptions(nil, true)
	handlerConfig := defaultHandlerConfig

	goHandler := NewGoHandler(&handlerConfig, options,
		func(event *types.Event) error {
			return nil
		}, func(event *types.Event) error {
			return nil
		})

	goHandler.exitFunction = func(i int) {
		exitStatus = i
	}
	goHandler.Execute()
	assert.Equal(t, 1, exitStatus)
}

func goHandlerExecuteUtil(t *testing.T, handlerConfig *PluginConfig, nilDefaults bool, eventFile string, cmdLineArgs []string,
	validationFunction func(*types.Event) error, executeFunction func(*types.Event) error,
	expectedValue1 interface{}, expectedValue2 interface{}, expectedValue3 interface{}, expectedValue4 interface{}, expectedValue5 interface{}) (int, string) {

	t.Helper()
	values := handlerValues{}
	options := getHandlerOptions(&values, nilDefaults)

	goHandler := NewGoHandler(handlerConfig, options, validationFunction, executeFunction)

	// Simulate the command line arguments if necessary
	if len(cmdLineArgs) > 0 {
		goHandler.cmd.SetArgs(cmdLineArgs)
	} else {
		goHandler.cmd.SetArgs([]string{})
	}

	goHandler.cmd.SilenceErrors = true
	goHandler.cmd.SilenceUsage = true

	// Replace stdin reader with file reader and exitFunction with our own so we can know the exit status
	var exitStatus int
	var errorStr = ""
	goHandler.eventReader = getFileReader(eventFile)
	goHandler.exitFunction = func(i int) {
		exitStatus = i
	}
	goHandler.errorLogFunction = func(format string, a ...interface{}) {
		errorStr = fmt.Sprintf(format, a...)
	}
	goHandler.Execute()
	assert.Equal(t, expectedValue1, values.arg1)
	assert.Equal(t, expectedValue2, values.arg2)
	assert.Equal(t, expectedValue3, values.arg3)
	assert.Equal(t, expectedValue4, len(values.arg4))
	assert.Equal(t, expectedValue5, len(values.arg5))

	return exitStatus, errorStr
}

// Test check override
func TestGoHandler_Execute_Check(t *testing.T) {
	for _, nilDefaults := range []bool{true, false} {
		var validateCalled, executeCalled bool
		clearEnvironment()
		exitStatus, _ := goHandlerExecuteUtil(t, &defaultHandlerConfig, nilDefaults, "test/event-check-override.json", nil,
			func(event *types.Event) error {
				validateCalled = true
				assert.NotNil(t, event)
				return nil
			}, func(event *types.Event) error {
				executeCalled = true
				assert.NotNil(t, event)
				return nil
			},
			"value-check1", uint64(1357), false, 2, 2)
		assert.Equal(t, 0, exitStatus)
		assert.True(t, validateCalled)
		assert.True(t, executeCalled)
	}
}

// Test check override with invalid value
func TestGoHandler_Execute_CheckInvalidValue(t *testing.T) {
	for _, nilDefaults := range []bool{true, false} {
		var validateCalled, executeCalled bool
		clearEnvironment()
		exitStatus, _ := goHandlerExecuteUtil(t, &defaultHandlerConfig, nilDefaults, "test/event-check-override-invalid-value.json", nil,
			func(event *types.Event) error {
				validateCalled = true
				assert.NotNil(t, event)
				return nil
			}, func(event *types.Event) error {
				executeCalled = true
				assert.NotNil(t, event)
				return nil
			},
			"value-check1", uint64(33333), false, 0, 0)
		assert.Equal(t, 1, exitStatus)
		assert.False(t, validateCalled)
		assert.False(t, executeCalled)
	}
}

// Test entity override
func TestGoHandler_Execute_Entity(t *testing.T) {
	for _, nilDefaults := range []bool{true, false} {
		var validateCalled, executeCalled bool
		clearEnvironment()
		exitStatus, _ := goHandlerExecuteUtil(t, &defaultHandlerConfig, nilDefaults, "test/event-entity-override.json", nil,
			func(event *types.Event) error {
				validateCalled = true
				assert.NotNil(t, event)
				return nil
			}, func(event *types.Event) error {
				executeCalled = true
				assert.NotNil(t, event)
				return nil
			},
			"value-entity1", uint64(2468), true, 0, 0)

		assert.Equal(t, 0, exitStatus)
		assert.True(t, validateCalled)
		assert.True(t, executeCalled)
	}
}

// Test entity override - invalid value
func TestGoHandler_Execute_EntityInvalidValue(t *testing.T) {
	for _, nilDefaults := range []bool{true, false} {
		var validateCalled, executeCalled bool
		clearEnvironment()
		exitStatus, _ := goHandlerExecuteUtil(t, &defaultHandlerConfig, nilDefaults, "test/event-entity-override-invalid-value.json", nil,
			func(event *types.Event) error {
				validateCalled = true
				assert.NotNil(t, event)
				return nil
			}, func(event *types.Event) error {
				executeCalled = true
				assert.NotNil(t, event)
				return nil
			},
			"value-entity1", uint64(33333), false, 0, 0)

		assert.Equal(t, 1, exitStatus)
		assert.False(t, validateCalled)
		assert.False(t, executeCalled)
	}
}

// Test environment
func TestGoHandler_Execute_Environment(t *testing.T) {
	for _, nilDefaults := range []bool{true, false} {
		var validateCalled, executeCalled bool
		clearEnvironment()
		_ = os.Setenv("ENV_1", "value-env1")
		_ = os.Setenv("ENV_2", "9753")
		_ = os.Setenv("ENV_3", "true")
		_ = os.Setenv("ENV_4", "this is a space delimited string slice")
		_ = os.Setenv("ENV_5", "this is a single string entry in string array")
		exitStatus, _ := goHandlerExecuteUtil(t, &defaultHandlerConfig, nilDefaults, "test/event-no-override.json", nil,
			func(event *types.Event) error {
				validateCalled = true
				assert.NotNil(t, event)
				return nil
			}, func(event *types.Event) error {
				executeCalled = true
				assert.NotNil(t, event)
				return nil
			},
			"value-env1", uint64(9753), true, 7, 1)
		assert.Equal(t, 0, exitStatus)
		assert.True(t, validateCalled)
		assert.True(t, executeCalled)
	}
}

// Test cmd line arguments
func TestGoHandler_Execute_CmdLineArgs(t *testing.T) {
	for _, nilDefaults := range []bool{true, false} {
		var validateCalled, executeCalled bool
		clearEnvironment()
		exitStatus, _ := goHandlerExecuteUtil(t, &defaultHandlerConfig, nilDefaults, "test/event-no-override.json", defaultCmdLineArgs,
			func(event *types.Event) error {
				validateCalled = true
				assert.NotNil(t, event)
				return nil
			}, func(event *types.Event) error {
				executeCalled = true
				assert.NotNil(t, event)
				return nil
			},
			"value-arg1", uint64(7531), false, 4, 2)
		assert.Equal(t, 0, exitStatus)
		assert.True(t, validateCalled)
		assert.True(t, executeCalled)
	}
}

// Test check priority - check override
func TestGoHandler_Execute_PriorityCheck(t *testing.T) {
	for _, nilDefaults := range []bool{true, false} {
		var validateCalled, executeCalled bool
		clearEnvironment()
		_ = os.Setenv("ENV_1", "value-env1")
		_ = os.Setenv("ENV_2", "9753")
		_ = os.Setenv("ENV_3", "true")
		exitStatus, _ := goHandlerExecuteUtil(t, &defaultHandlerConfig, nilDefaults, "test/event-check-entity-override.json", defaultCmdLineArgs,
			func(event *types.Event) error {
				validateCalled = true
				assert.NotNil(t, event)
				return nil
			}, func(event *types.Event) error {
				executeCalled = true
				assert.NotNil(t, event)
				return nil
			},
			"value-check1", uint64(1357), false, 4, 2)
		assert.Equal(t, 0, exitStatus)
		assert.True(t, validateCalled)
		assert.True(t, executeCalled)
	}
}

// Test next priority - entity override
func TestGoHandler_Execute_PriorityEntity(t *testing.T) {
	for _, nilDefaults := range []bool{true, false} {
		var validateCalled, executeCalled bool
		clearEnvironment()
		_ = os.Setenv("ENV_1", "value-env1")
		_ = os.Setenv("ENV_2", "9753")
		_ = os.Setenv("ENV_3", "true")
		exitStatus, _ := goHandlerExecuteUtil(t, &defaultHandlerConfig, nilDefaults, "test/event-entity-override.json", defaultCmdLineArgs,
			func(event *types.Event) error {
				validateCalled = true
				assert.NotNil(t, event)
				return nil
			}, func(event *types.Event) error {
				executeCalled = true
				assert.NotNil(t, event)
				return nil
			},
			"value-entity1", uint64(2468), true, 4, 2)
		assert.Equal(t, 0, exitStatus)
		assert.True(t, validateCalled)
		assert.True(t, executeCalled)
	}
}

// Test next priority - cmd line arguments
func TestGoHandler_Execute_PriorityCmdLine(t *testing.T) {
	for _, nilDefaults := range []bool{true, false} {
		var validateCalled, executeCalled bool
		clearEnvironment()
		_ = os.Setenv("ENV_1", "value-env1")
		_ = os.Setenv("ENV_2", "9753")
		_ = os.Setenv("ENV_3", "true")
		exitStatus, _ := goHandlerExecuteUtil(t, &defaultHandlerConfig, nilDefaults, "test/event-no-override.json", defaultCmdLineArgs,
			func(event *types.Event) error {
				validateCalled = true
				assert.NotNil(t, event)
				return nil
			}, func(event *types.Event) error {
				executeCalled = true
				assert.NotNil(t, event)
				return nil
			},
			"value-arg1", uint64(7531), false, 4, 2)
		assert.Equal(t, 0, exitStatus)
		assert.True(t, validateCalled)
		assert.True(t, executeCalled)
	}
}

// Test validation error
func TestGoHandler_Execute_ValidationError(t *testing.T) {
	for _, nilDefaults := range []bool{true, false} {
		var validateCalled, executeCalled bool
		clearEnvironment()
		exitStatus, errorStr := goHandlerExecuteUtil(t, &defaultHandlerConfig, nilDefaults, "test/event-no-override.json", defaultCmdLineArgs,
			func(event *types.Event) error {
				validateCalled = true
				assert.NotNil(t, event)
				return fmt.Errorf("validation error")
			}, func(event *types.Event) error {
				executeCalled = true
				return nil
			},
			"value-arg1", uint64(7531), false, 4, 2)
		assert.Equal(t, 1, exitStatus)
		assert.Contains(t, errorStr, "error validating input: validation error")
		assert.True(t, validateCalled)
		assert.False(t, executeCalled)
	}
}

// Test execute error
func TestGoHandler_Execute_ExecuteError(t *testing.T) {
	for _, nilDefaults := range []bool{true, false} {
		var validateCalled, executeCalled bool
		clearEnvironment()
		exitStatus, errorStr := goHandlerExecuteUtil(t, &defaultHandlerConfig, nilDefaults, "test/event-no-override.json", defaultCmdLineArgs,
			func(event *types.Event) error {
				validateCalled = true
				assert.NotNil(t, event)
				return nil
			}, func(event *types.Event) error {
				executeCalled = true
				assert.NotNil(t, event)
				return fmt.Errorf("execution error")
			},
			"value-arg1", uint64(7531), false, 4, 2)
		assert.Equal(t, 1, exitStatus)
		assert.Contains(t, errorStr, "error executing handler: execution error")
		assert.True(t, validateCalled)
		assert.True(t, executeCalled)
	}
}

// Test invalid event - no timestamp
func TestGoHandler_Execute_EventNoTimestamp(t *testing.T) {
	for _, nilDefaults := range []bool{true, false} {
		var validateCalled, executeCalled bool
		clearEnvironment()
		exitStatus, errorStr := goHandlerExecuteUtil(t, &defaultHandlerConfig, nilDefaults, "test/event-no-timestamp.json", defaultCmdLineArgs,
			func(event *types.Event) error {
				validateCalled = true
				assert.NotNil(t, event)
				return nil
			}, func(event *types.Event) error {
				executeCalled = true
				assert.NotNil(t, event)
				return nil
			},
			"value-arg1", uint64(7531), false, 4, 2)
		assert.Equal(t, 1, exitStatus)
		assert.Contains(t, errorStr, "timestamp is missing or must be greater than zero")
		assert.False(t, validateCalled)
		assert.False(t, executeCalled)
	}
}

// Test invalid event - timestamp 0
func TestGoHandler_Execute_EventTimestampZero(t *testing.T) {
	for _, nilDefaults := range []bool{true, false} {
		var validateCalled, executeCalled bool
		clearEnvironment()
		exitStatus, errorStr := goHandlerExecuteUtil(t, &defaultHandlerConfig, nilDefaults, "test/event-timestamp-zero.json", defaultCmdLineArgs,
			func(event *types.Event) error {
				validateCalled = true
				assert.NotNil(t, event)
				return nil
			}, func(event *types.Event) error {
				executeCalled = true
				assert.NotNil(t, event)
				return nil
			},
			"value-arg1", uint64(7531), false, 4, 2)
		assert.Equal(t, 1, exitStatus)
		assert.Contains(t, errorStr, "timestamp is missing or must be greater than zero")
		assert.False(t, validateCalled)
		assert.False(t, executeCalled)
	}
}

// Test invalid event - no entity
func TestGoHandler_Execute_EventNoEntity(t *testing.T) {
	for _, nilDefaults := range []bool{true, false} {
		var validateCalled, executeCalled bool
		clearEnvironment()
		exitStatus, errorStr := goHandlerExecuteUtil(t, &defaultHandlerConfig, nilDefaults, "test/event-no-entity.json", defaultCmdLineArgs,
			func(event *types.Event) error {
				validateCalled = true
				assert.NotNil(t, event)
				return nil
			}, func(event *types.Event) error {
				executeCalled = true
				assert.NotNil(t, event)
				return nil
			},
			"value-arg1", uint64(7531), false, 4, 2)
		assert.Equal(t, 1, exitStatus)
		assert.Contains(t, errorStr, "event must contain an entity")
		assert.False(t, validateCalled)
		assert.False(t, executeCalled)
	}
}

// Test invalid event - invalid entity
func TestGoHandler_Execute_EventInvalidEntity(t *testing.T) {
	for _, nilDefaults := range []bool{true, false} {
		var validateCalled, executeCalled bool
		clearEnvironment()
		exitStatus, errorStr := goHandlerExecuteUtil(t, &defaultHandlerConfig, nilDefaults, "test/event-invalid-entity.json", defaultCmdLineArgs,
			func(event *types.Event) error {
				validateCalled = true
				assert.NotNil(t, event)
				return nil
			}, func(event *types.Event) error {
				executeCalled = true
				assert.NotNil(t, event)
				return nil
			},
			"value-arg1", uint64(7531), false, 4, 2)
		assert.Equal(t, 1, exitStatus)
		assert.Contains(t, errorStr, "entity name must not be empty")
		assert.False(t, validateCalled)
		assert.False(t, executeCalled)
	}
}

// Test invalid event - no check
func TestGoHandler_Execute_EventNoCheck(t *testing.T) {
	for _, nilDefaults := range []bool{true, false} {
		var validateCalled, executeCalled bool
		clearEnvironment()
		exitStatus, errorStr := goHandlerExecuteUtil(t, &defaultHandlerConfig, nilDefaults, "test/event-no-check.json", defaultCmdLineArgs,
			func(event *types.Event) error {
				validateCalled = true
				assert.NotNil(t, event)
				return nil
			}, func(event *types.Event) error {
				executeCalled = true
				assert.NotNil(t, event)
				return nil
			},
			"value-arg1", uint64(7531), false, 4, 2)
		assert.Equal(t, 1, exitStatus)
		assert.Contains(t, errorStr, "event must contain a check or metrics")
		assert.False(t, validateCalled)
		assert.False(t, executeCalled)
	}
}

// Test invalid event - invalid check
func TestGoHandler_Execute_EventInvalidCheck(t *testing.T) {
	for _, nilDefaults := range []bool{true, false} {
		var validateCalled, executeCalled bool
		clearEnvironment()
		exitStatus, errorStr := goHandlerExecuteUtil(t, &defaultHandlerConfig, nilDefaults, "test/event-invalid-check.json", defaultCmdLineArgs,
			func(event *types.Event) error {
				validateCalled = true
				assert.NotNil(t, event)
				return nil
			}, func(event *types.Event) error {
				executeCalled = true
				assert.NotNil(t, event)
				return nil
			},
			"value-arg1", uint64(7531), false, 4, 2)
		assert.Equal(t, 1, exitStatus)
		assert.Contains(t, errorStr, "check name must not be empty")
		assert.False(t, validateCalled)
		assert.False(t, executeCalled)
	}
}

// Test unmarshalling error
func TestGoHandler_Execute_EventInvalidJson(t *testing.T) {
	for _, nilDefaults := range []bool{true, false} {
		var validateCalled, executeCalled bool
		clearEnvironment()
		exitStatus, errorStr := goHandlerExecuteUtil(t, &defaultHandlerConfig, nilDefaults, "test/event-invalid-json.json", defaultCmdLineArgs,
			func(event *types.Event) error {
				validateCalled = true
				assert.NotNil(t, event)
				return nil
			}, func(event *types.Event) error {
				executeCalled = true
				assert.NotNil(t, event)
				return nil
			},
			"value-arg1", uint64(7531), false, 4, 2)
		assert.Equal(t, 1, exitStatus)
		assert.Contains(t, errorStr, "Failed to unmarshal STDIN data: invalid character ':' after object key:value pair")
		assert.False(t, validateCalled)
		assert.False(t, executeCalled)
	}
}

// Test fail to read stdin
func TestGoHandler_Execute_ReaderError(t *testing.T) {
	for _, nilDefaults := range []bool{true, false} {
		var validateCalled, executeCalled bool
		clearEnvironment()
		exitStatus, errorStr := goHandlerExecuteUtil(t, &defaultHandlerConfig, nilDefaults, "test/event-invalid-json.json", defaultCmdLineArgs,
			func(event *types.Event) error {
				validateCalled = true
				assert.NotNil(t, event)
				return nil
			}, func(event *types.Event) error {
				executeCalled = true
				assert.NotNil(t, event)
				return nil
			},
			"value-arg1", uint64(7531), false, 4, 2)
		assert.Equal(t, 1, exitStatus)
		assert.Contains(t, errorStr, "Failed to unmarshal STDIN data: invalid character ':' after object key:value pair")
		assert.False(t, validateCalled)
		assert.False(t, executeCalled)
	}
}

// Test no keyspace
func TestGoHandler_Execute_NoKeyspace(t *testing.T) {
	for _, nilDefaults := range []bool{true, false} {
		var validateCalled, executeCalled bool
		clearEnvironment()
		handlerConfig := defaultHandlerConfig
		handlerConfig.Keyspace = ""
		exitStatus, _ := goHandlerExecuteUtil(t, &handlerConfig, nilDefaults, "test/event-check-entity-override.json", defaultCmdLineArgs,
			func(event *types.Event) error {
				validateCalled = true
				assert.NotNil(t, event)
				return nil
			}, func(event *types.Event) error {
				executeCalled = true
				assert.NotNil(t, event)
				return nil
			},
			"value-arg1", uint64(7531), false, 4, 2)
		assert.Equal(t, 0, exitStatus)
		assert.True(t, validateCalled)
		assert.True(t, executeCalled)
	}
}

func getHandlerOptions(values *handlerValues, nilDefaults bool) []*PluginConfigOption {
	option1 := defaultOption1
	option2 := defaultOption2
	option3 := defaultOption3
	option4 := defaultOption4
	option5 := defaultOption5
	if nilDefaults {
		option1.Default = nil
		option2.Default = nil
		option3.Default = nil
		option4.Default = nil
		option5.Default = nil
	}
	if values != nil {
		option1.Value = &values.arg1
		option2.Value = &values.arg2
		option3.Value = &values.arg3
		option4.Value = &values.arg4
		option5.Value = &values.arg5
	} else {
		option1.Value = nil
		option2.Value = nil
		option3.Value = nil
		option4.Value = nil
		option5.Value = nil
	}
	return []*PluginConfigOption{&option1, &option2, &option3, &option4, &option5}
}

func TestNewGoHandlerEnterprise(t *testing.T) {
	for _, nilDefaults := range []bool{true, false} {
		var exitStatus int
		values := &handlerValues{}
		options := getHandlerOptions(values, nilDefaults)
		goHandler := NewEnterpriseGoHandler(&defaultHandlerConfig, options, func(event *types.Event) error {
			return nil
		}, func(event *types.Event) error {
			return nil
		})
		assert.True(t, goHandler.enterprise)

		goHandler.exitFunction = func(i int) {
			exitStatus = i
		}
		goHandler.Execute()
		assert.Equal(t, 1, exitStatus)
	}
}

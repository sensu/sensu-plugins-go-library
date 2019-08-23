package sensu

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/sensu/sensu-go/types"
	"github.com/sensu/sensu-plugins-go-library/args"
	"io"
	"io/ioutil"
	"log"
	"os"
	"path"
	"strconv"
)

// GoPlugin defines the GoPlugin interface to be implemented by all types of plugins
type GoPlugin interface {
	Execute()
}

// PluginConfigOption defines an option to be read by the plugin on startup. An
// option can be passed using a command line argument, an environment variable or
// at for some plugin types using a configuration override from the Sensu event.
type PluginConfigOption struct {
	Value     interface{}
	Path      string
	Env       string
	Argument  string      // command line argument
	Shorthand string      // short command line argument
	Default   interface{} // default value
	Usage     string
}

// PluginConfig defines the base plugin configuration.
type PluginConfig struct {
	Name     string
	Short    string
	Timeout  uint64
	Keyspace string
}

// basePlugin defines the basic configuration to be used by all plugin types.
type basePlugin struct {
	config                 *PluginConfig
	options                []*PluginConfigOption
	sensuEvent             *types.Event
	eventReader            io.Reader
	pluginWorkflowFunction func([]string) (int, error)
	cmdArgs                *args.Args
	readEvent              bool
	eventMandatory         bool
	configurationOverrides bool
	exitStatus             int
	errorExitStatus        int
	exitFunction           func(int)
	errorLogFunction       func(format string, a ...interface{})
}

const (
	noExitStatus = 2147483647
)

func (goPlugin *basePlugin) readSensuEvent() error {
	eventJSON, err := ioutil.ReadAll(goPlugin.eventReader)
	if err != nil {
		if goPlugin.eventMandatory {
			return fmt.Errorf("Failed to read STDIN: %s", err)
		} else {
			// if event is not mandatory return without going any further
			return nil
		}
	}

	sensuEvent := &types.Event{}
	err = json.Unmarshal(eventJSON, sensuEvent)
	if err != nil {
		return fmt.Errorf("Failed to unmarshal STDIN data: %s", err)
	}

	if err = validateEvent(sensuEvent); err != nil {
		return err
	}

	goPlugin.sensuEvent = sensuEvent
	return nil
}

func (goPlugin *basePlugin) initPlugin() {
	goPlugin.cmdArgs = args.NewArgs(goPlugin.config.Name, goPlugin.config.Short, goPlugin.cobraExecuteFunction)
	goPlugin.exitFunction = os.Exit
	goPlugin.errorLogFunction = func(format string, a ...interface{}) {
		_, _ = fmt.Fprintf(os.Stderr, format, a)
	}
	goPlugin.exitStatus = noExitStatus
}

func (goPlugin *basePlugin) setupArguments() error {
	for _, option := range goPlugin.options {
		if option.Value == nil {
			return fmt.Errorf("Option value must not be nil for %s", option.Argument)
		}

		err := goPlugin.cmdArgs.SetVarP(option.Value, option.Argument, option.Shorthand, option.Env, option.Default, option.Usage)
		if err != nil {
			return fmt.Errorf("error setting up arguments: %s", err)
		}
	}

	return nil
}

// cobraExecuteFunction is called by the argument's execute. The configuration overrides will be processed if necessary
// and the pluginWorkflowFunction function executed
func (goPlugin *basePlugin) cobraExecuteFunction(args []string) error {
	// Read the Sensu event if required
	if goPlugin.readEvent {
		err := goPlugin.readSensuEvent()
		if err != nil {
			goPlugin.exitStatus = goPlugin.errorExitStatus
			return err
		}
	}

	// If there is an event process configuration overrides if necessary
	if goPlugin.sensuEvent != nil && goPlugin.configurationOverrides {
		err := configurationOverrides(goPlugin.config, goPlugin.options, goPlugin.sensuEvent)
		if err != nil {
			goPlugin.exitStatus = goPlugin.errorExitStatus
			return err
		}
	}

	exitStatus, err := goPlugin.pluginWorkflowFunction(args)
	if err != nil {
		fmt.Printf("Error executing plugin: %s", err)
	}
	goPlugin.exitStatus = exitStatus

	return err
}

func (goPlugin *basePlugin) Execute() {
	// Validate the arguments are set
	if goPlugin.cmdArgs == nil {
		goPlugin.errorLogFunction("Error executing %s: Arguments must be initialized\n", goPlugin.config.Name)
		goPlugin.exitFunction(goPlugin.errorExitStatus)
		return
	}

	err := goPlugin.setupArguments()
	if err != nil {
		goPlugin.errorLogFunction("Error executing %s: %s\n", goPlugin.config.Name, err)
		goPlugin.exitFunction(goPlugin.errorExitStatus)
		return
	}

	// This will call the pluginWorkflowFunction function which implements the custom logic for that type of plugin.
	err = goPlugin.cmdArgs.Execute()
	if err != nil {
		goPlugin.errorLogFunction("Error executing %s: %v\n", goPlugin.config.Name, err)
		if goPlugin.exitStatus != noExitStatus {
			goPlugin.exitFunction(goPlugin.exitStatus)
		} else {
			goPlugin.exitFunction(goPlugin.errorExitStatus)
		}
		return
	}

	goPlugin.exitFunction(goPlugin.exitStatus)
}

func validateEvent(event *types.Event) error {
	if event.Timestamp <= 0 {
		return errors.New("timestamp is missing or must be greater than zero")
	}

	return event.Validate()
}

func setOptionValue(option *PluginConfigOption, valueStr string) error {
	switch option.Value.(type) {
	case *string:
		strOptionValue, ok := option.Value.(*string)
		if ok {
			*strOptionValue = valueStr
		}
	case *uint64:
		uint64OptionPtrValue, ok := option.Value.(*uint64)
		if ok {
			parsedValue, err := strconv.ParseUint(valueStr, 10, 64)
			if err != nil {
				return fmt.Errorf("Error parsing %s into a uint64 for option %s", valueStr, option.Argument)
			}
			*uint64OptionPtrValue = parsedValue
		}
	case *uint32:
		uint32OptionPtrValue, ok := option.Value.(*uint32)
		if ok {
			parsedValue, err := strconv.ParseUint(valueStr, 10, 32)
			if err != nil {
				return fmt.Errorf("Error parsing %s into a uint32 for option %s", valueStr, option.Argument)
			}
			*uint32OptionPtrValue = uint32(parsedValue)
		}
	case *uint16:
		uint16OptionPtrValue, ok := option.Value.(*uint16)
		if ok {
			parsedValue, err := strconv.ParseUint(valueStr, 10, 16)
			if err != nil {
				return fmt.Errorf("Error parsing %s into a uint16 for option %s", valueStr, option.Argument)
			}
			*uint16OptionPtrValue = uint16(parsedValue)
		}
	case *int64:
		int64OptionPtrValue, ok := option.Value.(*int64)
		if ok {
			parsedValue, err := strconv.ParseInt(valueStr, 10, 64)
			if err != nil {
				return fmt.Errorf("Error parsing %s into a int64 for option %s", valueStr, option.Argument)
			}
			*int64OptionPtrValue = parsedValue
		}
	case *int32:
		int32OptionPtrValue, ok := option.Value.(*int32)
		if ok {
			parsedValue, err := strconv.ParseInt(valueStr, 10, 32)
			if err != nil {
				return fmt.Errorf("Error parsing %s into a uint32 for option %s", valueStr, option.Argument)
			}
			*int32OptionPtrValue = int32(parsedValue)
		}
	case *int16:
		int16OptionPtrValue, ok := option.Value.(*int16)
		if ok {
			parsedValue, err := strconv.ParseInt(valueStr, 10, 16)
			if err != nil {
				return fmt.Errorf("Error parsing %s into a uint16 for option %s", valueStr, option.Argument)
			}
			*int16OptionPtrValue = int16(parsedValue)
		}
	case *bool:
		boolOptionPtrValue, ok := option.Value.(*bool)
		if ok {
			parsedValue, err := strconv.ParseBool(valueStr)
			if err != nil {
				return fmt.Errorf("Error parsing %s into a bool for option %s", valueStr, option.Argument)
			}
			*boolOptionPtrValue = parsedValue
		}
	}
	return nil
}

func configurationOverrides(config *PluginConfig, options []*PluginConfigOption, event *types.Event) error {
	if config.Keyspace == "" {
		return nil
	}
	for _, opt := range options {
		if len(opt.Path) > 0 {
			// compile the Annotation keyspace to look for configuration overrides
			key := path.Join(config.Keyspace, opt.Path)
			switch {
			case len(event.Check.Annotations[key]) > 0:
				err := setOptionValue(opt, event.Check.Annotations[key])
				if err != nil {
					return err
				}
				log.Printf("Overriding default handler configuration with value of \"Check.Annotations.%s\" (\"%s\")\n",
					key, event.Check.Annotations[key])
			case len(event.Entity.Annotations[key]) > 0:
				err := setOptionValue(opt, event.Entity.Annotations[key])
				if err != nil {
					return err
				}
				log.Printf("Overriding default handler configuration with value of \"Entity.Annotations.%s\" (\"%s\")\n",
					key, event.Entity.Annotations[key])
			}
		}
	}
	return nil
}

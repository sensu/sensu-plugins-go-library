package sensu

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"path"
	"reflect"
	"strings"

	"github.com/sensu/sensu-go/types"
	"github.com/sensu/sensu-plugin-sdk/version"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// GoPlugin defines the GoPlugin interface to be implemented by all types of plugins
type GoPlugin interface {
	Execute()
}

// PluginConfigOption defines an option to be read by the plugin on startup. An
// option can be passed using a command line argument, an environment variable or
// at for some plugin types using a configuration override from the Sensu event.
type PluginConfigOption struct {
	// Value is the value to read the configured flag or environment variable into.
	// Pass a pointer to any value in your plugin in order to fill it in with the
	// data from a flag or environment variable. The parsing will be done with
	// a function supplied by viper. See the viper documentation for details on
	// how various data types are parsed.
	Value interface{}

	// Path is the path to the Sensu annotation to consult when parsing config.
	Path string

	// Env is the environment variable to consult when parsing config.
	Env string

	// Argument is the command line argument to consult when parsing config.
	Argument string

	// Shorthand is the shorthand command line argument to consult when parsing config.
	Shorthand string

	// Default is the default value of the config option.
	Default interface{}

	// Usage adds help context to the command-line flag.
	Usage string

	// If secret option do not copy Argument value into Default
	Secret bool

	// If array option set, treat StringSlice as StringArray, do not automatically parse as CSV space delimited
	Array bool
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
	cmd                    *cobra.Command
	readEvent              bool
	eventMandatory         bool
	eventValidation        bool
	configurationOverrides bool
	exitStatus             int
	errorExitStatus        int
	exitFunction           func(int)
	errorLogFunction       func(format string, a ...interface{})
}

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
	if goPlugin.eventValidation {
		if err = validateEvent(sensuEvent); err != nil {
			return err
		}
	}

	goPlugin.sensuEvent = sensuEvent
	return nil
}

func (p *basePlugin) initPlugin() error {
	p.cmd = &cobra.Command{
		Use:           p.config.Name,
		Short:         p.config.Short,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := viper.BindPFlags(cmd.Flags()); err != nil {
				return err
			}
			return p.cobraExecuteFunction(args)
		},
	}
	p.exitFunction = os.Exit
	p.errorLogFunction = func(format string, a ...interface{}) {
		_, _ = fmt.Fprintf(os.Stderr, format, a...)
	}

	p.cmd.AddCommand(&cobra.Command{
		Use:           "version",
		Short:         "Print the version number of this plugin",
		SilenceErrors: true,
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println(version.Version())
		},
	})

	return p.setupFlags(p.cmd)
}

func (p *basePlugin) setupFlags(cmd *cobra.Command) error {
	for _, opt := range p.options {
		if err := setupFlag(cmd, opt); err != nil {
			return err
		}
	}
	return nil
}

func setupFlag(cmd *cobra.Command, opt *PluginConfigOption) error {
	if len(opt.Argument) == 0 {
		return nil
	}
	err := viper.BindEnv(opt.Argument, opt.Env)
	if err != nil {
		return err
	}
	if opt.Value == nil {
		return errors.New("nil Value")
	}
	if reflect.TypeOf(opt.Value).Kind() != reflect.Ptr {
		return errors.New("Value is not a pointer")
	}
	value := reflect.Indirect(reflect.ValueOf(opt.Value))
	if opt.Default != nil {
		defaultType := reflect.TypeOf(opt.Default)
		valueType := value.Type()
		if t1, t2 := valueType.Kind(), defaultType.Kind(); t1 != t2 {
			return fmt.Errorf("Value type does not match Default type: %v != %v", t1, t2)
		}
		viper.SetDefault(opt.Argument, opt.Default)
	}
	switch kind := value.Type().Kind(); kind {
	case reflect.Bool:
		cmd.Flags().BoolVarP(opt.Value.(*bool), opt.Argument, opt.Shorthand, viper.GetBool(opt.Argument), opt.Usage)
	case reflect.Int:
		cmd.Flags().IntVarP(opt.Value.(*int), opt.Argument, opt.Shorthand, viper.GetInt(opt.Argument), opt.Usage)
	case reflect.Int32:
		cmd.Flags().Int32VarP(opt.Value.(*int32), opt.Argument, opt.Shorthand, viper.GetInt32(opt.Argument), opt.Usage)
	case reflect.Int64:
		cmd.Flags().Int64VarP(opt.Value.(*int64), opt.Argument, opt.Shorthand, viper.GetInt64(opt.Argument), opt.Usage)
	case reflect.Uint:
		cmd.Flags().UintVarP(opt.Value.(*uint), opt.Argument, opt.Shorthand, viper.GetUint(opt.Argument), opt.Usage)
	case reflect.Uint32:
		cmd.Flags().Uint32VarP(opt.Value.(*uint32), opt.Argument, opt.Shorthand, viper.GetUint32(opt.Argument), opt.Usage)
	case reflect.Uint64:
		cmd.Flags().Uint64VarP(opt.Value.(*uint64), opt.Argument, opt.Shorthand, viper.GetUint64(opt.Argument), opt.Usage)
	case reflect.Float32:
		cmd.Flags().Float32VarP(opt.Value.(*float32), opt.Argument, opt.Shorthand, float32(viper.GetFloat64(opt.Argument)), opt.Usage)
	case reflect.Float64:
		cmd.Flags().Float64VarP(opt.Value.(*float64), opt.Argument, opt.Shorthand, viper.GetFloat64(opt.Argument), opt.Usage)
	case reflect.Map:
		ptr, ok := opt.Value.(*map[string]string)
		if !ok {
			return fmt.Errorf("only pointer to map[string]string is allowed, not %v", kind)
		}
		cmd.Flags().StringToStringVarP(ptr, opt.Argument, opt.Shorthand, viper.GetStringMapString(opt.Argument), opt.Usage)
	case reflect.Slice:
		ptr, ok := opt.Value.(*[]string)
		if !ok {
			return fmt.Errorf("only pointer to []string is allowed, not %v", kind)
		}
		if opt.Array {
			// Treat this as a StringArrayVar do not automatically parse string as space delimited list
			// Grab the fallback envvar value as string via viper
			val := viper.GetString(opt.Argument)
			strSlice := []string{}
			// If the fallback value is non-empty, append it as first string in array
			if len(val) > 0 {
				strSlice = append(strSlice, val)
			}
			cmd.Flags().StringArrayVarP(ptr, opt.Argument, opt.Shorthand, strSlice, opt.Usage)
		} else {
			cmd.Flags().StringSliceVarP(ptr, opt.Argument, opt.Shorthand, viper.GetStringSlice(opt.Argument), opt.Usage)
		}
	case reflect.String:
		cmd.Flags().StringVarP(opt.Value.(*string), opt.Argument, opt.Shorthand, viper.GetString(opt.Argument), opt.Usage)
	default:
		return fmt.Errorf("invalid input type: %v", kind)
	}
	flag := cmd.Flags().Lookup(opt.Argument)
	// Set empty DefValue string if option is a secret
	// DefValue is only used for pflag usage message construction
	if opt.Secret {
		flag.DefValue = ""
	}
	return nil
}

// cobraExecuteFunction is called by the argument's execute. The configuration overrides will be processed if necessary
// and the pluginWorkflowFunction function executed
func (p *basePlugin) cobraExecuteFunction(args []string) error {
	// Read the Sensu event if required
	if p.readEvent {
		err := p.readSensuEvent()
		if err != nil {
			p.exitStatus = p.errorExitStatus
			return err
		}
	}

	// If there is an event process configuration overrides if necessary
	if p.sensuEvent != nil && p.configurationOverrides {
		err := configurationOverrides(p.config, p.options, p.sensuEvent)
		if err != nil {
			p.exitStatus = p.errorExitStatus
			return err
		}
	}

	exitStatus, err := p.pluginWorkflowFunction(args)
	p.exitStatus = exitStatus

	return err
}

func (p *basePlugin) Execute() {
	// Validate the cmd is set
	if p.cmd == nil {
		p.errorLogFunction("Error executing %s: Arguments must be initialized\n", p.config.Name)
		p.exitFunction(p.errorExitStatus)
	}

	if err := p.cmd.Execute(); err != nil {
		p.errorLogFunction("Error executing %s: %v\n", p.config.Name, err)
	}

	p.exitFunction(p.exitStatus)
}

func validateEvent(event *types.Event) error {
	if event.Timestamp <= 0 {
		return errors.New("timestamp is missing or must be greater than zero")
	}

	return event.Validate()
}

func setOptionValue(opt *PluginConfigOption, valueStr string) error {
	optVal := reflect.Indirect(reflect.ValueOf(opt.Value))
	if typ := optVal.Type(); typ.Kind() == reflect.Slice {
		if err := json.Unmarshal([]byte(valueStr), &opt.Value); err == nil {
			return nil
		}
		if typ.Elem().Kind() == reflect.String {
			empty := []string{}
			optVal.Set(reflect.Append(reflect.ValueOf(empty), reflect.ValueOf(valueStr)))
			return nil
		}
	}
	if optVal.Type().Kind() == reflect.String {
		optVal.Set(reflect.ValueOf(valueStr))
		return nil
	}
	return json.Unmarshal([]byte(valueStr), &opt.Value)
}

func configurationOverrides(config *PluginConfig, options []*PluginConfigOption, event *types.Event) error {
	if config.Keyspace == "" {
		return nil
	}
	for _, opt := range options {
		if len(opt.Path) > 0 {
			// compile the Annotation keyspace to look for configuration overrides
			key := path.Join(config.Keyspace, opt.Path)
			downcase := strings.ToLower(key)
			keys := []string{downcase, key}
			for _, key := range keys {
				switch {
				case event.Check != nil && len(event.Check.Annotations[key]) > 0:
					err := setOptionValue(opt, event.Check.Annotations[key])
					if err != nil {
						return err
					}
					log.Printf("Overriding default handler configuration with value of \"Check.Annotations.%s\" (\"%s\")\n",
						key, event.Check.Annotations[key])
				case event.Entity != nil && len(event.Entity.Annotations[key]) > 0:
					err := setOptionValue(opt, event.Entity.Annotations[key])
					if err != nil {
						return err
					}
					log.Printf("Overriding default handler configuration with value of \"Entity.Annotations.%s\" (\"%s\")\n",
						key, event.Entity.Annotations[key])
				}
			}
		}
	}
	return nil
}

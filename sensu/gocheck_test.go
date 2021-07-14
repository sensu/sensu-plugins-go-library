package sensu

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/sensu/sensu-go/types"
)

type checkValues struct {
	arg1 string
	arg2 uint64
	arg3 bool
	arg4 []string
	arg5 []string
}

var (
	defaultCheckConfig = PluginConfig{
		Name:     "TestCheck",
		Short:    "Short Description",
		Timeout:  10,
		Keyspace: "sensu.io/plugins/segp/config",
	}

	checkOption1 = defaultOption1
	checkOption2 = defaultOption2
	checkOption3 = defaultOption3
	checkOption4 = defaultOption4
	checkOption5 = defaultOption5
)

func TestNewGoCheck(t *testing.T) {
	values := &checkValues{}
	options := getCheckOptions(values)
	goCheck := NewGoCheck(&defaultCheckConfig, options, func(_ *types.Event) (int, error) {
		return 0, nil
	}, func(_ *types.Event) (int, error) {
		return 0, nil
	}, false)
	var exitStatus = -99
	goCheck.exitFunction = func(i int) {
		exitStatus = i
	}
	goCheck.Execute()
	assert.Equal(t, 0, exitStatus)

	assert.NotNil(t, goCheck)
	assert.NotNil(t, goCheck.options)
	assert.Equal(t, options, goCheck.options)
	assert.NotNil(t, goCheck.config)
	assert.Equal(t, &defaultCheckConfig, goCheck.config)
	assert.NotNil(t, goCheck.validationFunction)
	assert.NotNil(t, goCheck.executeFunction)
	assert.NotNil(t, goCheck.cmd)
	assert.Nil(t, goCheck.sensuEvent)
	assert.Equal(t, os.Stdin, goCheck.eventReader)
}

func getCheckOptions(values *checkValues) []*PluginConfigOption {
	option1 := checkOption1
	option2 := checkOption2
	option3 := checkOption3
	option4 := checkOption4
	option5 := checkOption5
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

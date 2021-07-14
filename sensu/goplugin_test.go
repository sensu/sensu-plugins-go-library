package sensu

import (
	"io"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSetOptionValue_String(t *testing.T) {
	finalValue := ""
	option := defaultOption1
	option.Value = &finalValue
	err := setOptionValue(&option, "abc")
	assert.Nil(t, err)
	assert.Equal(t, "abc", finalValue)
}

func TestSetOptionValue_EmptyString(t *testing.T) {
	finalValue := ""
	option := defaultOption1
	option.Value = &finalValue
	err := setOptionValue(&option, "")
	assert.Nil(t, err)
	assert.Equal(t, "", finalValue)
}

func TestSetOptionValue_Slice(t *testing.T) {
	finalValue := []string{"def"}
	option := defaultOption1
	option.Value = &finalValue
	err := setOptionValue(&option, "abc")
	assert.Nil(t, err)
	assert.Equal(t, []string{"abc"}, finalValue)
}

func TestSetOptionValue_EmptySlice(t *testing.T) {
	finalValue := []string{"def"}
	option := defaultOption1
	option.Value = &finalValue
	err := setOptionValue(&option, "")
	assert.Nil(t, err)
	assert.Equal(t, []string{""}, finalValue)
}

func TestSetOptionValue_StringSliceType(t *testing.T) {
	type stringSlice []string
	finalValue := stringSlice{"def"}
	option := defaultOption1
	option.Value = &finalValue
	err := setOptionValue(&option, "abc")
	assert.Nil(t, err)
	assert.Equal(t, stringSlice{"abc"}, finalValue)
}

func TestSetOptionValue_StringArrayType(t *testing.T) {
	type stringArray []string
	finalValue := stringArray{"def"}
	option := defaultOption1
	option.Value = &finalValue
	option.Array = true
	err := setOptionValue(&option, "abc")
	assert.Nil(t, err)
	assert.Equal(t, stringArray{"abc"}, finalValue)
}

func TestSetOptionValue_JSONArrayStringSlice(t *testing.T) {
	var finalValue []string
	option := defaultOption1
	option.Value = &finalValue
	err := setOptionValue(&option, `["abc","def"]`)
	assert.Nil(t, err)
	assert.Equal(t, []string{"abc", "def"}, finalValue)
}

func TestSetOptionValue_JSONArrayStringArray(t *testing.T) {
	var finalValue []string
	option := defaultOption1
	option.Value = &finalValue
	option.Array = true
	err := setOptionValue(&option, `["abc","def"]`)
	assert.Nil(t, err)
	assert.Equal(t, []string{"abc", "def"}, finalValue)
}

func TestSetOptionValue_ValidUint64(t *testing.T) {
	var finalValue uint64
	option := defaultOption1
	option.Value = &finalValue
	err := setOptionValue(&option, "123")
	assert.Nil(t, err)
	assert.Equal(t, uint64(123), finalValue)
}

func TestSetOptionValue_InvalidUint64(t *testing.T) {
	var finalValue uint64
	option := defaultOption1
	option.Value = &finalValue
	err := setOptionValue(&option, "abc")
	assert.NotNil(t, err)
	assert.Equal(t, uint64(0), finalValue)
}

func TestSetOptionValue_TrueBool(t *testing.T) {
	var finalValue bool
	option := defaultOption1
	option.Value = &finalValue
	err := setOptionValue(&option, "true")
	assert.Nil(t, err)
	assert.Equal(t, true, finalValue)
}

func TestSetOptionValue_FalseBool(t *testing.T) {
	finalValue := true
	option := defaultOption1
	option.Value = &finalValue
	err := setOptionValue(&option, "false")
	assert.Nil(t, err)
	assert.Equal(t, false, finalValue)
}

func TestSetOptionValue_Int64(t *testing.T) {
	var value int64
	option := defaultOption1
	option.Value = &value
	assert.NoError(t, setOptionValue(&option, "42"))
	assert.Equal(t, value, int64(42))
}

func TestSetOptionValue_InvalidBool(t *testing.T) {
	var finalValue bool
	option := defaultOption1
	option.Value = &finalValue
	err := setOptionValue(&option, "yes")
	assert.NotNil(t, err)
	assert.Equal(t, false, finalValue)
}

func getFileReader(file string) io.Reader {
	reader, _ := os.Open(file)
	return reader
}

func clearEnvironment() {
	_ = os.Unsetenv("ENV_1")
	_ = os.Unsetenv("ENV_2")
	_ = os.Unsetenv("ENV_3")
	_ = os.Unsetenv("ENV_4")
	_ = os.Unsetenv("ENV_5")
}

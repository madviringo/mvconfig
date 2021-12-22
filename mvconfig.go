/**

MV Config - this module allows us to configure variables from the Environment Variables or a properties file.

You pass in a struct to the function and it will try and get the values from the Environemnt, or a properties file.

You can define default values (in the tags) and also a Critical value - which means an error will be thrown if the
value isn't in the Env or Props
`def:"123" crit:"true"`

You don't need to pass a properties file - it will automatically try to load app.properties, but will ignore it if the
file doesn't exist.

The name of the field will be used as the ENVIRONMENT variable - however - you can define the variable name in the tags.
`mvenv:"ENV_NAME"`

You can also pass in a PREFIX to be used on all the variables in the struct.  The PREFIX will be added the front of the
names with an '_'.  So if PREFIX = ZITOBOX - then the var names will be ZITOBOX_NAME


Usage
-----

//Environment or props file

IntValue=1
STRVAL=HelloThere
BOOLVAL=true

type EnvVars struct {
	IntValue int
	StringValue string `mvenv:"STRVAL" def:"hello"`
    IsSafe bool `mvenv:"BOOLVAL" crit:"true"`
}

envs := EnvVars{}
err := mvconfig.LoadVariables(&envs)
if err != nil {
	// manage error
}

fmt.Println(envs)


*/
package mvconfig

import (
	"errors"
	"fmt"
	"github.com/magiconair/properties"
	"os"
	"reflect"
	"strconv"
	"strings"
)

type envTags struct {
	Name       string
	Critical   bool
	HasDefault bool
	Default    string
}

func LoadVariables(envStruct interface{}) error {
	return loadVariables(envStruct, "", "app.properties")
}

func LoadVariablesWithProps(envStruct interface{}, fileName string) error {
	return loadVariables(envStruct, "", fileName)
}

func LoadVariablesWithPrefix(envStruct interface{}, prefix string) error {
	return loadVariables(envStruct, prefix, "app.properties")
}

func LoadVariablesWithPrefixAndProps(envStruct interface{}, prefix string, fileName string) error {
	return loadVariables(envStruct, prefix, fileName)
}

func loadVariables(envStruct interface{}, prefix string, fileName string) error {
	// Load the properties file
	props, err := properties.LoadFile(fileName, properties.UTF8)
	if err != nil {
		props = nil
	}

	err = manageFields(envStruct, props, prefix)
	return err
}

/**
Get all of the tag values in this field
*/
func getTags(f reflect.StructField) (envTags, error) {
	e := envTags{}
	e.Name = f.Name
	e.Critical = false
	e.HasDefault = false
	e.Default = ""

	if value, ok := f.Tag.Lookup("mvenv"); ok {
		e.Name = value
	}
	if crit, ok := f.Tag.Lookup("crit"); ok {
		crit = strings.ToLower(crit)
		if crit == "true" || crit == "y" || crit == "t" {
			e.Critical = true
		}
	}

	if defval, ok := f.Tag.Lookup("def"); ok {
		e.HasDefault = true
		e.Default = defval
	}

	return e, nil
}

/**
Get the values from the environment, properties or default
*/
func getEnvValue(eTags envTags, props *properties.Properties, prefix string) (string, error, bool) {

	name := eTags.Name
	if prefix != "" {
		name = prefix + "_" + name
	}

	// Try from the environment variables first
	if value, ok := os.LookupEnv(name); ok {
		fmt.Println(name, value)
		return value, nil, true
	}

	// If not in the environment - check the properties file
	if props != nil {
		if value, ok := props.Get(name); ok {
			return value, nil, true
		}
	}

	// If not in the properties - check the default value
	if eTags.HasDefault {
		return eTags.Default, nil, true
	}

	// If no default val - if its critical return an error
	if eTags.Critical {
		return "", errors.New("Critical Config field " + name + " missing from the environment"), false
	}

	// Otherwise - do nothing
	return "", nil, false
}

func manageFields(envVar interface{}, props *properties.Properties, prefix string) error {

	e := reflect.ValueOf(envVar).Elem()
	t := e.Type()

	for i := 0; i < t.NumField(); i++ {

		eTags, err := getTags(t.Field(i))
		if err == nil {
			// Need to lookup the field value
			if value, err, ok := getEnvValue(eTags, props, prefix); ok {
				fld := e.FieldByName(t.Field(i).Name)
				if fld.CanSet() {
					if e.Field(i).Kind() == reflect.String {
						fld.SetString(value)
					} else if e.Field(i).Kind() == reflect.Int ||
						e.Field(i).Kind() == reflect.Int8 ||
						e.Field(i).Kind() == reflect.Int32 ||
						e.Field(i).Kind() == reflect.Int64 {
						val, err := strconv.Atoi(value)
						if err != nil {
							return errors.New("Error converting field " + eTags.Name + " to int")
						}
						fld.SetInt(int64(val))
					} else if e.Field(i).Kind() == reflect.Bool {
						val, err := strconv.ParseBool(value)
						if err != nil {
							return errors.New("Error converting field " + eTags.Name + " to bool")
						}
						fld.SetBool(val)
					} else if e.Field(i).Kind() == reflect.Float32 ||
						e.Field(i).Kind() == reflect.Float32 {
						val, err := strconv.ParseFloat(value, 64)
						if err != nil {
							return errors.New("Error converting field " + eTags.Name + " to float")
						}
						fld.SetFloat(val)
					}
				}
			} else if err != nil {
				return err
			}
		}
	}
	return nil
}

package config

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"

	"github.com/mcuadros/go-defaults"
	"github.com/spf13/viper"

	"github.com/spy16/pgbase/strutils"
)

// Load loads configurations into the given structPtr.
func Load(structPtr interface{}, opts ...Option) error {
	defs, err := extractConfigDefs(structPtr)
	if err != nil {
		return err
	}

	l := &viperLoader{
		viper:       viper.New(),
		intoPtr:     structPtr,
		configs:     defs,
		useDefaults: true,
	}

	for _, opt := range opts {
		if err := opt(l); err != nil {
			return err
		}
	}

	return l.load()
}

type viperLoader struct {
	viper       *viper.Viper
	configs     []configDef
	intoPtr     interface{}
	confFile    string
	confName    string
	useEnv      bool
	envPrefix   string
	useDefaults bool
}

func (l *viperLoader) load() error {
	v := l.viper

	if l.useDefaults {
		defaults.SetDefaults(l.intoPtr)
	}

	for _, cfg := range l.configs {
		v.SetDefault(cfg.Key, cfg.Default)
	}

	if l.useEnv {
		// for transforming app.host to app_host
		v.SetEnvKeyReplacer(strings.NewReplacer(".", "_", "-", "_"))
		v.SetEnvPrefix(l.envPrefix)
		v.AutomaticEnv()
		for _, cfg := range l.configs {
			if err := v.BindEnv(cfg.Key); err != nil {
				return err
			}
		}
	}

	if l.confFile != "" {
		v.SetConfigFile(l.confFile)
		if err := v.ReadInConfig(); err != nil {
			return err
		}
	} else {
		if l.confName == "" {
			l.confName = "config"
		}
		v.AddConfigPath("./")
		v.AddConfigPath(getExecPath())
		v.SetConfigName(l.confName)
		_ = v.ReadInConfig()
	}

	return v.Unmarshal(l.intoPtr)
}

type configDef struct {
	Key     string      `json:"key"`
	Default interface{} `json:"default"`
}

func extractConfigDefs(structPtr interface{}) ([]configDef, error) {
	rv := reflect.ValueOf(structPtr)

	if err := ensureStructPtr(rv); err != nil {
		return nil, err
	}

	return readRecursive(deref(rv), "")
}

func readRecursive(rv reflect.Value, rootKey string) ([]configDef, error) {
	rt := rv.Type()

	var acc []configDef
	for i := 0; i < rv.NumField(); i++ {
		ft := rt.Field(i)
		fv := deref(rv.Field(i))

		key := strings.SplitN(ft.Tag.Get("mapstructure"), ",", 2)[0]
		if key == "" {
			key = strutils.SnakeCase(ft.Name)
		}

		if rootKey != "" {
			key = fmt.Sprintf("%s.%s", rootKey, key)
		}

		if fv.Kind() == reflect.Struct {
			nestedConfigs, err := readRecursive(fv, key)
			if err != nil {
				return nil, err
			}
			acc = append(acc, nestedConfigs...)
		} else {
			acc = append(acc, configDef{
				Key:     key,
				Default: fv.Interface(),
			})
		}
	}

	return acc, nil
}

func deref(rv reflect.Value) reflect.Value {
	if rv.Kind() == reflect.Ptr {
		rv = reflect.Indirect(rv)
	}
	return rv
}

func ensureStructPtr(value reflect.Value) error {
	if value.Kind() != reflect.Ptr {
		return fmt.Errorf("need a pointer to struct, not '%s'", value.Kind())
	} else {
		value = reflect.Indirect(value)
		if value.Kind() != reflect.Struct {
			return fmt.Errorf("need a pointer to struct, not pointer to '%s'", value.Kind())
		}
	}
	return nil
}

func getExecPath() string {
	execPath, err := os.Executable()
	if err != nil {
		return ""
	}
	return filepath.Dir(execPath)
}

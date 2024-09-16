package packaged

import (
	"fmt"
	"os"
	"reflect"
	"strconv"
	"strings"
	"time"
)

func parseEnv[T any](m *envManager, key string, parser func(string) (T, error)) (T, error) {
	key = strings.ToUpper(key)
	var zero T
	value, exists := m.GetEnv(key)
	if !exists {
		return zero, fmt.Errorf("packaged: environment variable %s not found", key)
	}

	return parser(value)
}

func (m *envManager) assign(v reflect.Value, prefix string) error {
	if v.Kind() != reflect.Ptr || v.Elem().Kind() != reflect.Struct {
		return fmt.Errorf("packaged: config must be a pointer to a struct")
	}
	v = v.Elem()

	for i := 0; i < v.NumField(); i++ {
		field := v.Type().Field(i)
		fieldValue := v.Field(i)

		envKey := field.Tag.Get("env")
		if envKey == "" {
			envKey = field.Name
		}
		if prefix != "" {
			envKey = prefix + "_" + envKey
		}

		if fieldValue.Kind() == reflect.Struct {
			err := m.assign(fieldValue.Addr(), envKey)
			if err != nil {
				return err
			}
			continue
		}

		value, exists := m.GetEnv(envKey)
		if !exists {
			if field.Tag.Get("required") == "true" {
				return fmt.Errorf("packaged: required environment variable %s not found", envKey)
			}
			continue
		}

		err := setField(fieldValue, value)
		if err != nil {
			return fmt.Errorf("packaged: error setting field %s: %v", field.Name, err)
		}
	}

	return nil
}

func setField(field reflect.Value, value string) error {
	switch field.Kind() {
	case reflect.String:
		field.SetString(value)
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		intValue, err := strconv.ParseInt(value, 10, 64)
		if err != nil {
			return err
		}
		field.SetInt(intValue)
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		uintValue, err := strconv.ParseUint(value, 10, 64)
		if err != nil {
			return err
		}
		field.SetUint(uintValue)
	case reflect.Float32, reflect.Float64:
		floatValue, err := strconv.ParseFloat(value, 64)
		if err != nil {
			return err
		}
		field.SetFloat(floatValue)
	case reflect.Bool:
		boolValue, err := strconv.ParseBool(value)
		if err != nil {
			return err
		}
		field.SetBool(boolValue)
	case reflect.Slice:
		return setSliceField(field, value)
	default:
		return fmt.Errorf("packaged: unsupported type: %v", field.Kind())
	}
	return nil
}

func setSliceField(field reflect.Value, value string) error {
	sliceValues := strings.Split(value, ",")
	slice := reflect.MakeSlice(field.Type(), 0, len(sliceValues))

	for _, val := range sliceValues {
		val = strings.TrimSpace(val)
		newElem := reflect.New(field.Type().Elem()).Elem()
		err := setField(newElem, val)
		if err != nil {
			return fmt.Errorf("packaged: error setting slice element: %v", err)
		}
		slice = reflect.Append(slice, newElem)
	}

	field.Set(slice)
	return nil
}

func splitEnv(s string) (string, string, bool) {
	if i := strings.Index(s, "="); i >= 0 {
		return strings.ToUpper(s[:i]), strings.ToUpper(s[i+1:]), true
	}
	return "", "", false
}

type EnvManager interface {
	GetEnv(key string) (string, bool)
	GetEnvInt(key string) (int, error)
	GetEnvFloat(key string) (float64, error)
	GetEnvBool(key string) (bool, error)
	GetEnvTime(key string, layout string) (time.Time, error)
	GetEnvDuration(key string) (time.Duration, error)
	Assign(dest any) error
}

type envManager struct {
	registry map[string]string
}

func (m *envManager) GetEnv(key string) (string, bool) {
	value, exists := m.registry[key]
	return value, exists
}

func (m *envManager) GetEnvInt(key string) (int, error) {
	return parseEnv[int](m, key, func(s string) (int, error) {
		return strconv.Atoi(s)
	})
}

func (m *envManager) GetEnvFloat(key string) (float64, error) {
	return parseEnv[float64](m, key, func(s string) (float64, error) {
		return strconv.ParseFloat(s, 64)
	})
}

func (m *envManager) GetEnvBool(key string) (bool, error) {
	return parseEnv[bool](m, key, strconv.ParseBool)
}

func (m *envManager) GetEnvTime(key string, layout string) (time.Time, error) {
	return parseEnv[time.Time](m, key, func(s string) (time.Time, error) {
		return time.Parse(layout, s)
	})
}

func (m *envManager) GetEnvDuration(key string) (time.Duration, error) {
	return parseEnv[time.Duration](m, key, time.ParseDuration)
}

func (m *envManager) Assign(dest any) error {
	return m.assign(reflect.ValueOf(dest), "")

}

func lookupPrefix(prefix string) *envManager {
	mgr := &envManager{
		registry: make(map[string]string),
	}
	prefix = strings.ToUpper(prefix)
	for _, env := range os.Environ() {
		key, value, ok := splitEnv(env)
		if !ok {
			continue
		}
		if strings.HasPrefix(key, prefix) {
			mgr.registry[key] = value
		}
	}
	return mgr
}

package config

import (
	"os"
	"reflect"
	"strconv"
	"strings"
)

type Config struct {
	HOST               string
	PORT               int
	ALLOWED_EXTENSIONS []string
	MAX_TASKS          int
	MAX_FILES_IN_ZIP   int
}

func GetConfig() *Config {
	cfg := &Config{
		HOST:               "localhost",
		PORT:               8080,
		ALLOWED_EXTENSIONS: []string{"jpeg", "pdf"},
		MAX_TASKS:          3,
		MAX_FILES_IN_ZIP:   3,
	}

	val := reflect.ValueOf(cfg).Elem()
	typ := reflect.TypeOf(cfg).Elem()
	for i := range val.NumField() {
		name := typ.Field(i).Name
		env := os.Getenv(name)
		if env != "" {
			if name == "ALLOWED_EXTENSIONS" {
				spl := strings.Split(env, ",")
				var newExtensions []string
				newExtensions = append(newExtensions, spl...)
				extensions := reflect.ValueOf(newExtensions)
				val.FieldByName(name).Set(extensions)
				continue
			}
			value, err := strconv.Atoi(env)
			if err != nil {
				val.FieldByName(name).SetString(env)
				continue
			}
			val.FieldByName(name).SetInt(int64(value))
		}
	}
	return cfg
}

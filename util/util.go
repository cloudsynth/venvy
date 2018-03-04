package util

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/BurntSushi/toml"
	"github.com/mitchellh/go-homedir"
	"gopkg.in/go-playground/validator.v9"
	"html/template"
	"os"
	"path"
	"regexp"
	"sync"
)

func PathExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func UnmarshalEmpty(data []byte, v interface{}) error {
	if len(data) == 0 {
		return nil
	}
	return json.Unmarshal(data, v)
}

func StringTemplate(tmplName, tmpl string, data interface{}) (string, error) {
	parsedTemplate, err := template.New(tmplName).Parse(tmpl)
	if err != nil {
		return "", err
	}
	var out bytes.Buffer
	err = parsedTemplate.Execute(&out, data)
	if err != nil {
		return "", err
	}
	return out.String(), nil
}

// Convert toml to json and pass json to parser, this lets us json partial parsing techniques for the sum module.config type
func TomlToJson(data []byte) ([]byte, error) {
	var target interface{}
	err := toml.Unmarshal(data, &target)
	if err != nil {
		return nil, err
	}
	return json.Marshal(&target)
}

func FindPathInAncestors(start string, pathToFind string) (string, error) {
	if start == "" {
		var err error
		start, err = os.Getwd()
		if err != nil {
			return "", err
		}
	}
	if _, err := os.Stat(path.Join(start, pathToFind)); err == nil {
		return start, nil
	}
	if start == "/" || start == "." {
		return "", fmt.Errorf("not found")
	}
	return FindPathInAncestors(path.Dir(start), pathToFind)
}

func MustExpandPath(path string) string {
	expanded, err := homedir.Expand(path)
	if err != nil {
		panic(fmt.Errorf("unable to expand Path %s, check $HOME", path))
	}
	return expanded
}

var rootValidator *validator.Validate
var setupValidator sync.Once
var CleanNameRe = regexp.MustCompile(`^[a-z0-9_\-]+$`)

func ValidateStruct(data interface{}) error {
	// Not done in init to gurantee init order until the project is split into packages
	setupValidator.Do(func() {
		rootValidator = validator.New()
		rootValidator.RegisterValidation("cleanName", func(fl validator.FieldLevel) bool {
			fieldStr := fl.Field().String()
			return CleanNameRe.MatchString(fieldStr)
		})
	})
	return rootValidator.Struct(data)
}


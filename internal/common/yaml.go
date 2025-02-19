package common

import (
	"gopkg.in/yaml.v3"
	"io"
	"os"
)

func YamlLoad(path string, val interface{}) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()
	y, err := io.ReadAll(f)
	if err != nil {
		return err
	}
	err = yaml.Unmarshal(y, val)
	if err != nil {
		return err
	}
	return nil
}

func YamlStore(path string, val interface{}) error {
	y, err := yaml.Marshal(val)
	if err != nil {
		return err
	}
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = f.Write(y)
	if err != nil {
		return err
	}
	return nil
}

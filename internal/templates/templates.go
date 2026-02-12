package templates

import "embed"

//go:embed *.tmpl
var Templates embed.FS

func GetTemplate(name string) (string, error) {
	data, err := Templates.ReadFile(name)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

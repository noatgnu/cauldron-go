package models

type RegistryAuthor struct {
	ID    int    `json:"id"`
	Name  string `json:"name"`
	Email string `json:"email,omitempty"`
}

type RegistryCategory struct {
	ID          int    `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
}

type RegistryTag struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

type RegistryRuntime struct {
	ID     int    `json:"id"`
	Plugin string `json:"plugin"`
	Type   string `json:"type"`
	Script string `json:"script"`
}

type RegistryInput struct {
	ID          int     `json:"id"`
	Plugin      string  `json:"plugin"`
	Name        string  `json:"name"`
	Label       string  `json:"label"`
	Type        string  `json:"type"`
	Required    bool    `json:"required"`
	Default     string  `json:"default,omitempty"`
	Description string  `json:"description,omitempty"`
	Placeholder string  `json:"placeholder,omitempty"`
	Accept      string  `json:"accept,omitempty"`
	Multiple    bool    `json:"multiple"`
	SourceFile  string  `json:"sourceFile,omitempty"`
	Min         float64 `json:"min,omitempty"`
	Max         float64 `json:"max,omitempty"`
	Step        float64 `json:"step,omitempty"`
}

type RegistryOutput struct {
	ID          int    `json:"id"`
	Plugin      string `json:"plugin"`
	Name        string `json:"name"`
	Path        string `json:"path"`
	Type        string `json:"type"`
	Description string `json:"description,omitempty"`
	Format      string `json:"format,omitempty"`
}

type RegistryPlugin struct {
	ID          string            `json:"id"`
	Name        string            `json:"name"`
	Description string            `json:"description"`
	Version     string            `json:"version"`
	Author      *RegistryAuthor   `json:"author"`
	Category    *RegistryCategory `json:"category"`
	Icon        string            `json:"icon,omitempty"`
	Repository  string            `json:"repository,omitempty"`
	CreatedAt   string            `json:"created_at"`
	UpdatedAt   string            `json:"updated_at"`
	Tags        []RegistryTag     `json:"tags,omitempty"`
	Runtime     *RegistryRuntime  `json:"runtime"`
	Inputs      []RegistryInput   `json:"inputs"`
	Outputs     []RegistryOutput  `json:"outputs"`
}

type RegistryPluginListResponse struct {
	Count    int              `json:"count"`
	Next     *string          `json:"next"`
	Previous *string          `json:"previous"`
	Results  []RegistryPlugin `json:"results"`
}

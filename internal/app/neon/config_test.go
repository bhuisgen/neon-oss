package neon

import (
	"os"
	"path"
	"testing"
)

func TestLoadConfig_YAML(t *testing.T) {
	name := path.Join(t.TempDir(), "test.yaml")
	data := `
store:
  storage:
    memory:

server:
  listeners:
    default:
      test:

  sites:
    main:
      default:
        middlewares:
        handler:
`
	if err := os.WriteFile(name, []byte(data), 0600); err != nil {
		t.Error(err)
	}
	CONFIG_FILE = name

	tests := []struct {
		name    string
		want    *config
		wantErr bool
	}{
		{
			name: "default",
			want: &config{
				Store: &configStore{
					Config: map[string]interface{}{
						"storage": map[string]interface{}{
							"memory": map[string]interface{}{},
						},
					},
				},
				Fetcher: &configFetcher{},
				Loader:  &configLoader{},
				Server: &configServer{
					Config: map[string]interface{}{
						"listeners": map[string]interface{}{
							"default": map[string]interface{}{
								"test": map[string]interface{}{},
							},
						},
						"sites": map[string]interface{}{
							"main": map[string]interface{}{
								"default": map[string]interface{}{
									"middlewares": map[string]interface{}{},
									"handler":     map[string]interface{}{},
								},
							},
						},
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := LoadConfig()
			if (err != nil) != tt.wantErr {
				t.Errorf("LoadConfig() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
		})
	}
}

func TestLoadConfig_TOML(t *testing.T) {
	name := path.Join(t.TempDir(), "test.toml")
	data := `
[store.storage.memory]

[server.listeners.default.local]
listenAddr = "0.0.0.0"
listenPort = 8080

[server.sites.main]
listeners = ["default"]

[server.sites.main.routes.default]
`
	if err := os.WriteFile(name, []byte(data), 0600); err != nil {
		t.Error(err)
	}
	CONFIG_FILE = name

	tests := []struct {
		name    string
		want    *config
		wantErr bool
	}{
		{
			name: "default",
			want: &config{
				Store: &configStore{
					Config: map[string]interface{}{
						"storage": map[string]interface{}{
							"memory": map[string]interface{}{},
						},
					},
				},
				Fetcher: &configFetcher{},
				Loader:  &configLoader{},
				Server: &configServer{
					Config: map[string]interface{}{
						"listeners": map[string]interface{}{
							"default": map[string]interface{}{
								"test": map[string]interface{}{},
							},
						},
						"sites": map[string]interface{}{
							"main": map[string]interface{}{
								"default": map[string]interface{}{
									"middlewares": map[string]interface{}{},
									"handler":     map[string]interface{}{},
								},
							},
						},
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := LoadConfig()
			if (err != nil) != tt.wantErr {
				t.Errorf("LoadConfig() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
		})
	}
}

func TestLoadConfig_JSON(t *testing.T) {
	name := path.Join(t.TempDir(), "test.json")
	data := `
{
  "store": {
    "storage": {
      "memory": {}
    }
  },
  "server": {
    "listeners": {
      "default": {
        "local": {
          "listenAddr": "0.0.0.0",
          "listenPort": 8080
        }
      }
    },
    "sites": {
      "main": {
        "listeners": [
          "default"
        ],
        "routes": {
          "default": {}
        }
      }
    }
  }
}
`
	if err := os.WriteFile(name, []byte(data), 0600); err != nil {
		t.Error(err)
	}
	CONFIG_FILE = name

	tests := []struct {
		name    string
		want    *config
		wantErr bool
	}{
		{
			name: "default",
			want: &config{
				Store: &configStore{
					Config: map[string]interface{}{
						"storage": map[string]interface{}{
							"memory": map[string]interface{}{},
						},
					},
				},
				Fetcher: &configFetcher{},
				Loader:  &configLoader{},
				Server: &configServer{
					Config: map[string]interface{}{
						"listeners": map[string]interface{}{
							"default": map[string]interface{}{
								"test": map[string]interface{}{},
							},
						},
						"sites": map[string]interface{}{
							"main": map[string]interface{}{
								"default": map[string]interface{}{
									"middlewares": map[string]interface{}{},
									"handler":     map[string]interface{}{},
								},
							},
						},
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := LoadConfig()
			if (err != nil) != tt.wantErr {
				t.Errorf("LoadConfig() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
		})
	}
}

func TestGenerateConfig_YAML(t *testing.T) {
	name := path.Join(t.TempDir(), "test.yaml")
	CONFIG_FILE = name

	type args struct {
		syntax   string
		template string
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "default",
			args: args{
				syntax:   "yaml",
				template: "default",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := GenerateConfig(tt.args.syntax, tt.args.template); (err != nil) != tt.wantErr {
				t.Errorf("GenerateConfig() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestGenerateConfig_TOML(t *testing.T) {
	name := path.Join(t.TempDir(), "test.toml")
	CONFIG_FILE = name

	type args struct {
		syntax   string
		template string
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "default",
			args: args{
				syntax:   "toml",
				template: "default",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := GenerateConfig(tt.args.syntax, tt.args.template); (err != nil) != tt.wantErr {
				t.Errorf("GenerateConfig() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestGenerateConfig_JSON(t *testing.T) {
	name := path.Join(t.TempDir(), "test.json")
	CONFIG_FILE = name

	type args struct {
		syntax   string
		template string
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "default",
			args: args{
				syntax:   "json",
				template: "default",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := GenerateConfig(tt.args.syntax, tt.args.template); (err != nil) != tt.wantErr {
				t.Errorf("GenerateConfig() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

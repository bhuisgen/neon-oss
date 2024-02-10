package neon

import (
	"os"
	"path"
	"testing"
)

func TestLoadConfig_YAML(t *testing.T) {
	name := path.Join(t.TempDir(), "test.yaml")
	data := `
app:
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
	t.Setenv("CONFIG_FILE", name)

	tests := []struct {
		name    string
		want    map[string]interface{}
		wantErr bool
	}{
		{
			name: "default",
			want: map[string]interface{}{
				"store": map[string]interface{}{
					"storage": map[string]interface{}{
						"memory": map[string]interface{}{},
					},
				},
				"fetcher": map[string]interface{}{},
				"loader":  map[string]interface{}{},
				"server": map[string]interface{}{
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

func TestGenerateConfig(t *testing.T) {
	name := path.Join(t.TempDir(), "test.yaml")
	t.Setenv("CONFIG_FILE", name)

	type args struct {
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
				template: "default",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := GenerateConfig(tt.args.template); (err != nil) != tt.wantErr {
				t.Errorf("GenerateConfig() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

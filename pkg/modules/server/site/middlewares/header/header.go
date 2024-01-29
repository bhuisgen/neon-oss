package header

import (
	"errors"
	"log/slog"
	"net/http"
	"regexp"

	"github.com/mitchellh/mapstructure"

	"github.com/bhuisgen/neon/pkg/core"
	"github.com/bhuisgen/neon/pkg/module"
)

// headerMiddleware implements the header middleware.
type headerMiddleware struct {
	config  *headerMiddlewareConfig
	logger  *slog.Logger
	regexps []*regexp.Regexp
}

// headerMiddlewareConfig implements the header middleware configuration.
type headerMiddlewareConfig struct {
	Rules []HeaderRule `mapstructure:"rules"`
}

// HeaderRule implements a header rule.
type HeaderRule struct {
	Path string            `mapstructure:"path"`
	Set  map[string]string `mapstructure:"set"`
	Last bool              `mapstructure:"last"`
}

const (
	headerModuleID module.ModuleID = "server.site.middleware.header"
)

// init initializes the module.
func init() {
	module.Register(headerMiddleware{})
}

// ModuleInfo returns the module information.
func (m headerMiddleware) ModuleInfo() module.ModuleInfo {
	return module.ModuleInfo{
		ID: headerModuleID,
		NewInstance: func() module.Module {
			return &headerMiddleware{}
		},
	}
}

// Init initializes the middleware.
func (m *headerMiddleware) Init(config map[string]interface{}, logger *slog.Logger) error {
	m.logger = logger

	if err := mapstructure.Decode(config, &m.config); err != nil {
		m.logger.Error("Failed to parse configuration")
		return err
	}

	var errInit bool

	for index, rule := range m.config.Rules {
		if rule.Path == "" {
			m.logger.Error("Missing option or value", "rule", index+1, "option", "Path")
			errInit = true
			continue
		}
		re, err := regexp.Compile(rule.Path)
		if err != nil {
			m.logger.Error("Invalid regular expression", "rule", index+1, "option", "Path", "value", rule.Path)
			errInit = true
			continue
		} else {
			m.regexps = append(m.regexps, re)
		}
		for key := range rule.Set {
			if key == "" {
				m.logger.Error("Invalid key", "rule", index+1, "option", "Set", "value", key)
				errInit = true
				continue
			}
		}
	}

	if errInit {
		return errors.New("init error")
	}

	return nil
}

// Register registers the middleware.
func (m *headerMiddleware) Register(site core.ServerSite) error {
	err := site.RegisterMiddleware(m.Handler)
	if err != nil {
		return err
	}

	return nil
}

// Start starts the middleware.
func (m *headerMiddleware) Start() error {
	return nil
}

// Stop stops the middleware.
func (m *headerMiddleware) Stop() {
}

// Handler implements the middleware handler.
func (m *headerMiddleware) Handler(next http.Handler) http.Handler {
	f := func(w http.ResponseWriter, r *http.Request) {
		for index, regexp := range m.regexps {
			if regexp.MatchString(r.URL.Path) {
				for k, v := range m.config.Rules[index].Set {
					w.Header().Set(k, v)
				}
				if m.config.Rules[index].Last {
					break
				}
			}
		}

		next.ServeHTTP(w, r)
	}

	return http.HandlerFunc(f)
}

var _ core.ServerSiteMiddlewareModule = (*headerMiddleware)(nil)

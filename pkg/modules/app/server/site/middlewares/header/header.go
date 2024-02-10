package header

import (
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"regexp"

	"github.com/mitchellh/mapstructure"

	"github.com/bhuisgen/neon/pkg/core"
	"github.com/bhuisgen/neon/pkg/log"
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
	headerModuleID module.ModuleID = "app.server.site.middleware.header"
)

// init initializes the package.
func init() {
	module.Register(headerMiddleware{})
}

// ModuleInfo returns the module information.
func (m headerMiddleware) ModuleInfo() module.ModuleInfo {
	return module.ModuleInfo{
		ID: headerModuleID,
		NewInstance: func() module.Module {
			return &headerMiddleware{
				logger: slog.New(log.NewHandler(os.Stderr, string(headerModuleID), nil)),
			}
		},
	}
}

// Init initializes the middleware.
func (m *headerMiddleware) Init(config map[string]interface{}) error {
	if err := mapstructure.Decode(config, &m.config); err != nil {
		m.logger.Error("Failed to parse configuration", "err", err)
		return fmt.Errorf("parse config: %v", err)
	}

	var errConfig bool

	for index, rule := range m.config.Rules {
		if rule.Path == "" {
			m.logger.Error("Missing option or value", "rule", index+1, "option", "Path")
			errConfig = true
			continue
		}
		re, err := regexp.Compile(rule.Path)
		if err != nil {
			m.logger.Error("Invalid regular expression", "rule", index+1, "option", "Path", "value", rule.Path)
			errConfig = true
			continue
		} else {
			m.regexps = append(m.regexps, re)
		}
		for key := range rule.Set {
			if key == "" {
				m.logger.Error("Invalid key", "rule", index+1, "option", "Set", "value", key)
				errConfig = true
				continue
			}
		}
	}

	if errConfig {
		return errors.New("config")
	}

	return nil
}

// Register registers the middleware.
func (m *headerMiddleware) Register(site core.ServerSite) error {
	if err := site.RegisterMiddleware(m.Handler); err != nil {
		return fmt.Errorf("register middleware: %v", err)
	}

	return nil
}

// Start starts the middleware.
func (m *headerMiddleware) Start() error {
	return nil
}

// Stop stops the middleware.
func (m *headerMiddleware) Stop() error {
	return nil
}

// Handler implements the middleware handler.
func (m *headerMiddleware) Handler(next http.Handler) http.Handler {
	fn := func(w http.ResponseWriter, r *http.Request) {
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

	return http.HandlerFunc(fn)
}

var _ core.ServerSiteMiddlewareModule = (*headerMiddleware)(nil)

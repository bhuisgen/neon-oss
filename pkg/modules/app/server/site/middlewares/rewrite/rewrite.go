package rewrite

import (
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"regexp"
	"strings"

	"github.com/mitchellh/mapstructure"

	"github.com/bhuisgen/neon/pkg/core"
	"github.com/bhuisgen/neon/pkg/log"
	"github.com/bhuisgen/neon/pkg/module"
)

// rewriteMiddleware implements the rewrite middleware.
type rewriteMiddleware struct {
	config  *rewriteMiddlewareConfig
	logger  *slog.Logger
	regexps []*regexp.Regexp
}

// rewriteMiddlewareConfig implements the rewrite middleware configuration.
type rewriteMiddlewareConfig struct {
	Rules []RewriteRule `mapstructure:"rules"`
}

// RewriteRule implements a rewrite rule.
type RewriteRule struct {
	Path        string  `mapstructure:"path"`
	Replacement string  `mapstructure:"replacement"`
	Flag        *string `mapstructure:"flag"`
	Last        bool    `mapstructure:"last"`
}

const (
	rewriteModuleID module.ModuleID = "app.server.site.middleware.rewrite"

	rewriteRuleFlagRedirect  string = "redirect"
	rewriteRuleFlagPermanent string = "permanent"
)

// init initializes the package.
func init() {
	module.Register(rewriteMiddleware{})
}

// ModuleInfo returns the module information.
func (m rewriteMiddleware) ModuleInfo() module.ModuleInfo {
	return module.ModuleInfo{
		ID: rewriteModuleID,
		NewInstance: func() module.Module {
			return &rewriteMiddleware{
				logger: slog.New(log.NewHandler(os.Stderr, string(rewriteModuleID), nil)),
			}
		},
	}
}

// Init initializes the middleware.
func (m *rewriteMiddleware) Init(config map[string]interface{}) error {
	if err := mapstructure.Decode(config, &m.config); err != nil {
		m.logger.Error("Failed to parse configuration", "err", err)
		return fmt.Errorf("parse config: %v", err)
	}

	var errConfig bool

	for index, rule := range m.config.Rules {
		if rule.Path == "" {
			m.logger.Error("Missing option or value", "rule", index+1, "option", "Path")
			errConfig = true
		} else {
			re, err := regexp.Compile(rule.Path)
			if err != nil {
				m.logger.Error("Invalid regular expression", "rule", index+1, "option", "Path", "value", rule.Path)
				errConfig = true
			} else {
				m.regexps = append(m.regexps, re)
			}
		}
		if rule.Replacement == "" {
			m.logger.Error("Missing option or value", "rule", index+1, "option", "Replacement")
			errConfig = true
		}
		if rule.Flag != nil {
			switch *rule.Flag {
			case rewriteRuleFlagPermanent:
			case rewriteRuleFlagRedirect:
			default:
				m.logger.Error("Invalid value '%s'", "rule", index+1, "option", "Flag", "value", *rule.Flag)
				errConfig = true
			}
		}
	}

	if errConfig {
		return errors.New("config")
	}

	return nil
}

// Register registers the middleware.
func (m *rewriteMiddleware) Register(site core.ServerSite) error {
	if err := site.RegisterMiddleware(m.Handler); err != nil {
		return fmt.Errorf("register middleware: %v", err)
	}

	return nil
}

// Start starts the middleware.
func (m *rewriteMiddleware) Start() error {
	return nil
}

// Stop stops the middleware.
func (m *rewriteMiddleware) Stop() error {
	return nil
}

// Handler implements the middleware handler.
func (m *rewriteMiddleware) Handler(next http.Handler) http.Handler {
	fn := func(w http.ResponseWriter, r *http.Request) {
		var rewrite bool
		var path string = r.URL.Path
		var status int = http.StatusFound
		var redirect bool
		for index, regexp := range m.regexps {
			if regexp.MatchString(path) {
				rewrite = true
				path = m.config.Rules[index].Replacement

				if strings.HasPrefix(path, "http://") || strings.HasPrefix(path, "https://") {
					redirect = true
				}

				if m.config.Rules[index].Flag != nil {
					switch *m.config.Rules[index].Flag {
					case rewriteRuleFlagRedirect:
						status = http.StatusFound
						redirect = true
					case rewriteRuleFlagPermanent:
						status = http.StatusMovedPermanently
						redirect = true
					}
				}

				if m.config.Rules[index].Last {
					break
				}
			}
		}

		if rewrite {
			if redirect {
				http.Redirect(w, r, path, status)
				return
			}
			r.URL.Path = path
		}

		next.ServeHTTP(w, r)
	}

	return http.HandlerFunc(fn)
}

var _ core.ServerSiteMiddlewareModule = (*rewriteMiddleware)(nil)

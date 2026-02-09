package swaggerui

import (
	"net/http"
	"os"
	"path/filepath"

	"github.com/gin-gonic/gin"
)

// Minimal swagger UI without codegen, using docs/openapi.yaml.
func Register(r *gin.Engine) {
	r.GET("/openapi.yaml", func(c *gin.Context) {
		if p, ok := findUp("docs/openapi.yaml", 10); ok {
			c.File(p)
			return
		}
		c.JSON(http.StatusNotFound, gin.H{"error": "openapi.yaml not found on disk"})
	})

	r.GET("/docs", func(c *gin.Context) {
		c.Header("Content-Type", "text/html; charset=utf-8")
		c.String(http.StatusOK, swaggerHTML)
	})
}

func findUp(rel string, maxDepth int) (string, bool) {
	if maxDepth <= 0 {
		maxDepth = 6
	}
	dir, err := os.Getwd()
	if err != nil {
		return "", false
	}
	for i := 0; i <= maxDepth; i++ {
		p := filepath.Join(dir, rel)
		if _, err := os.Stat(p); err == nil {
			return p, true
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return "", false
}

const swaggerHTML = `<!doctype html>
<html>
  <head>
    <meta charset="utf-8" />
    <meta name="viewport" content="width=device-width, initial-scale=1" />
    <title>Sarbon API — Документация</title>
    <link rel="stylesheet" href="https://unpkg.com/swagger-ui-dist@5/swagger-ui.css" />
    <style>
      .topbar { display: none; }
    </style>
  </head>
  <body>
    <div id="swagger-ui"></div>
    <script src="https://unpkg.com/swagger-ui-dist@5/swagger-ui-bundle.js"></script>
    <script>
      window.onload = () => {
        const LS_PREFIX = 'sarbon_auth_';
        const K = {
          DeviceTypeHeader: LS_PREFIX + 'DeviceTypeHeader',
          LanguageHeader: LS_PREFIX + 'LanguageHeader',
          ClientTokenHeader: LS_PREFIX + 'ClientTokenHeader',
          UserTokenHeader: LS_PREFIX + 'UserTokenHeader',
        };

        function getLS(key, defVal) {
          const v = localStorage.getItem(key);
          if (v === null || v === undefined || v === '') return defVal;
          return v;
        }
        function setLS(key, val) {
          try { localStorage.setItem(key, val); } catch(e) {}
        }

        const SarbonAuthPlugin = function() {
          return {
            wrapComponents: {
              apiKeyAuth: function(Original, system) {
                return function(props) {
                  const React = system.React;
                  const Row = system.getComponent('Row');
                  const Col = system.getComponent('Col');
                  const Markdown = system.getComponent('Markdown', true);

                  const name = props.name;
                  const schema = props.schema;
                  const authorized = props.authorized;

                  const currentVal = (authorized && authorized.getIn([name, 'value'])) || '';

                  const onChangeProxy = function(value) {
                    setLS(K[name] || (LS_PREFIX + name), value);
                    if (props.onChange) {
                      props.onChange({ name: name, schema: schema, value: value });
                    }
                  };

                  // Render select for device/language; keep default UI for others.
                  if (name === 'DeviceTypeHeader' || name === 'LanguageHeader') {
                    const options = (name === 'DeviceTypeHeader')
                      ? ['web', 'ios', 'android']
                      : ['ru', 'uz', 'en', 'tr', 'zh'];

                    const value = currentVal || getLS(K[name], options[0]);

                    return React.createElement(
                      'div',
                      { className: 'auth-container' },
                      React.createElement('h4', null, name || schema.get('name'), ' \u00A0(apiKey)'),
                      React.createElement(
                        Row,
                        null,
                        React.createElement(
                          Col,
                          null,
                          React.createElement('div', null,
                            React.createElement('div', { style: { marginBottom: '6px' } },
                              React.createElement('b', null, 'Name:'), ' ', schema.get('name')
                            ),
                            React.createElement('div', { style: { marginBottom: '6px' } },
                              React.createElement('b', null, 'In:'), ' ', schema.get('in')
                            ),
                            React.createElement('div', { style: { marginBottom: '6px' } },
                              React.createElement('b', null, 'Value:')
                            ),
                            React.createElement(
                              'select',
                              {
                                value: value,
                                onChange: function(e) { onChangeProxy(e.target.value); },
                                style: { width: '100%', padding: '8px', borderRadius: '4px' }
                              },
                              options.map(function(o) {
                                return React.createElement('option', { key: o, value: o }, o);
                              })
                            ),
                            schema.get('description')
                              ? React.createElement('div', { style: { marginTop: '8px' } },
                                  React.createElement(Markdown, { source: schema.get('description') })
                                )
                              : null
                          )
                        )
                      )
                    );
                  }

                  // For normal apiKey inputs: keep original but persist to localStorage on change
                  const originalOnChange = props.onChange;
                  const nextProps = Object.assign({}, props, {
                    onChange: function(newState) {
                      if (newState && newState.name) {
                        setLS(K[newState.name] || (LS_PREFIX + newState.name), (newState.value || '').toString());
                      }
                      if (originalOnChange) originalOnChange(newState);
                    }
                  });

                  return React.createElement(Original, nextProps);
                }
              }
            }
          }
        };

        window.ui = SwaggerUIBundle({
          url: '/openapi.yaml',
          dom_id: '#swagger-ui',
          deepLinking: true,
          persistAuthorization: true,
          docExpansion: 'none',
          defaultModelsExpandDepth: -1,
          plugins: [SarbonAuthPlugin],
          requestInterceptor: (req) => {
            // Failsafe: always inject required base headers from localStorage.
            // This guarantees headers are sent even if Swagger UI didn't apply Authorize properly.
            req.headers = req.headers || {};
            const d = getLS(K.DeviceTypeHeader, 'web');
            const l = getLS(K.LanguageHeader, 'ru');
            const ct = getLS(K.ClientTokenHeader, '');
            const ut = getLS(K.UserTokenHeader, '');
            if (d) req.headers['X-Device-Type'] = d;
            if (l) req.headers['X-Language'] = l;
            if (ct) req.headers['X-Client-Token'] = ct;
            if (ut) req.headers['X-User-Token'] = ut;
            return req;
          }
        });

        // Auto-apply defaults from localStorage on page load (so refresh keeps headers)
        try { window.ui.preauthorizeApiKey('DeviceTypeHeader', getLS(K.DeviceTypeHeader, 'web')); } catch(e) {}
        try { window.ui.preauthorizeApiKey('LanguageHeader', getLS(K.LanguageHeader, 'ru')); } catch(e) {}
        try { window.ui.preauthorizeApiKey('ClientTokenHeader', getLS(K.ClientTokenHeader, '')); } catch(e) {}
        const ut = getLS(K.UserTokenHeader, '');
        if (ut) {
          try { window.ui.preauthorizeApiKey('UserTokenHeader', ut); } catch(e) {}
        }
      };
    </script>
  </body>
</html>`


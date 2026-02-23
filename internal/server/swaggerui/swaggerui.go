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

      /* Always white background */
      html, body { background: #fff; }

      /* Top menu */
      body { padding-top: 58px; }
      .sarbon-topmenu {
        position: fixed;
        top: 0;
        left: 0;
        right: 0;
        height: 58px;
        display: flex;
        align-items: center;
        justify-content: center;
        gap: 10px;
        padding: 0 14px;
        background: rgba(255,255,255,.92);
        backdrop-filter: blur(10px);
        border-bottom: 1px solid rgba(0,0,0,.08);
        z-index: 9999;
      }
      .sarbon-topmenu .brand {
        font-family: ui-sans-serif, system-ui, -apple-system, Segoe UI, Roboto, Helvetica, Arial;
        font-weight: 700;
        letter-spacing: .2px;
        margin-right: 8px;
        color: #111827;
      }
      .sarbon-topmenu .btn {
        appearance: none;
        border: 1px solid rgba(0,0,0,.12);
        background: #fff;
        color: #111827;
        border-radius: 999px;
        padding: 10px 14px;
        font-size: 14px;
        line-height: 1;
        cursor: pointer;
        transition: background .15s ease, border-color .15s ease, box-shadow .15s ease;
      }
      .sarbon-topmenu .btn:hover { background: #f9fafb; }
      .sarbon-topmenu .btn.active {
        background: #111827;
        color: #fff;
        border-color: #111827;
        box-shadow: 0 6px 18px rgba(17,24,39,.18);
      }

      /* Keep content width comfortable */
      .swagger-ui .wrapper { max-width: 1240px; }
    </style>
  </head>
  <body>
    <div class="sarbon-topmenu" role="navigation" aria-label="API groups">
      <div class="brand">Sarbon API</div>
      <button class="btn" data-group="drivers">Drivers Mobile</button>
      <button class="btn" data-group="dispatchers">Freelance Dispatchers</button>
      <button class="btn" data-group="admin">Admin</button>
    </div>
    <div id="swagger-ui"></div>
    <script src="https://unpkg.com/swagger-ui-dist@5/swagger-ui-bundle.js"></script>
    <script>
      window.onload = () => {
        const LS_PREFIX = 'sarbon_auth_';
        const DOCS_GROUP_KEY = LS_PREFIX + 'docs_group';
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

        function tagName(t) {
          if (!t) return '';
          if (typeof t === 'string') return t;
          if (t.get && typeof t.get === 'function') return t.get('name') || '';
          if (t.name) return t.name;
          return String(t);
        }

        const TAG_ORDER = [
          'Drivers / Auth',
          'Drivers / Registration',
          'Drivers / KYC',
          'Drivers / Profile',
          'Freelance Dispatchers / Auth',
          'Freelance Dispatchers / Registration',
          'Freelance Dispatchers / Profile',
          'Admin / Auth',
          'Admin / Companies',
          'Reference',
        ];
        function tagIndex(t) {
          const n = tagName(t);
          const i = TAG_ORDER.indexOf(n);
          return i === -1 ? 999 : i;
        }

        function normalizeGroup(g) {
          if (g === 'drivers' || g === 'dispatchers' || g === 'admin') return g;
          return 'drivers';
        }

        function getInitialGroup() {
          // Allow quick switching by query string (?group=drivers|dispatchers|admin)
          try {
            const qs = new URLSearchParams(window.location.search || '');
            const qg = qs.get('group');
            if (qg) return normalizeGroup(qg);
          } catch(e) {}
          return normalizeGroup(getLS(DOCS_GROUP_KEY, 'drivers'));
        }

        function isTagInGroup(tag, group) {
          if (!tag) return false;
          if (group === 'drivers') return tag.startsWith('Drivers /');
          if (group === 'dispatchers') return tag.startsWith('Freelance Dispatchers /');
          if (group === 'admin') return tag.startsWith('Admin /');
          return true;
        }

        function applyGroupFilter(group) {
          const sections = document.querySelectorAll('#swagger-ui .opblock-tag-section');
          sections.forEach((sec) => {
            const tagBtn = sec.querySelector('.opblock-tag');
            const t = (tagBtn && tagBtn.textContent ? tagBtn.textContent : '').trim();
            sec.style.display = isTagInGroup(t, group) ? '' : 'none';
          });
        }

        function setActiveMenu(group) {
          document.querySelectorAll('.sarbon-topmenu .btn[data-group]').forEach((b) => {
            b.classList.toggle('active', b.getAttribute('data-group') === group);
          });
        }

        function setGroup(group) {
          const g = normalizeGroup(group);
          setLS(DOCS_GROUP_KEY, g);
          setActiveMenu(g);
          applyGroupFilter(g);
        }

        window.ui = SwaggerUIBundle({
          url: '/openapi.yaml',
          dom_id: '#swagger-ui',
          deepLinking: true,
          persistAuthorization: true,
          docExpansion: 'none',
          defaultModelsExpandDepth: -1,
          tagsSorter: (a, b) => {
            const ai = tagIndex(a);
            const bi = tagIndex(b);
            if (ai !== bi) return ai - bi;
            const an = tagName(a).toLowerCase();
            const bn = tagName(b).toLowerCase();
            return an.localeCompare(bn);
          },
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

        // Menu bindings + persist group selection
        const initialGroup = getInitialGroup();
        setActiveMenu(initialGroup);
        document.querySelectorAll('.sarbon-topmenu .btn[data-group]').forEach((btn) => {
          btn.addEventListener('click', () => setGroup(btn.getAttribute('data-group')));
        });

        // Swagger UI renders async; re-apply filter when DOM changes.
        let filterTimer = null;
        const mo = new MutationObserver(() => {
          if (filterTimer) clearTimeout(filterTimer);
          filterTimer = setTimeout(() => applyGroupFilter(getLS(DOCS_GROUP_KEY, initialGroup)), 30);
        });
        const root = document.getElementById('swagger-ui');
        if (root) mo.observe(root, { childList: true, subtree: true });

        // Auto-apply defaults from localStorage on page load (so refresh keeps headers)
        try { window.ui.preauthorizeApiKey('DeviceTypeHeader', getLS(K.DeviceTypeHeader, 'web')); } catch(e) {}
        try { window.ui.preauthorizeApiKey('LanguageHeader', getLS(K.LanguageHeader, 'ru')); } catch(e) {}
        try { window.ui.preauthorizeApiKey('ClientTokenHeader', getLS(K.ClientTokenHeader, '')); } catch(e) {}
        const ut = getLS(K.UserTokenHeader, '');
        if (ut) {
          try { window.ui.preauthorizeApiKey('UserTokenHeader', ut); } catch(e) {}
        }

        // Initial filter apply (after first paint)
        setTimeout(() => setGroup(initialGroup), 0);
      };
    </script>
  </body>
</html>`


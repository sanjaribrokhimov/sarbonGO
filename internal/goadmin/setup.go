package goadmin

import (
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"html/template"

	_ "github.com/GoAdminGroup/go-admin/adapter/gin"
	_ "github.com/GoAdminGroup/go-admin/modules/db/drivers/postgres"

	"github.com/GoAdminGroup/go-admin/engine"
	"github.com/GoAdminGroup/go-admin/modules/config"
	"github.com/GoAdminGroup/go-admin/modules/language"
	"github.com/GoAdminGroup/themes/adminlte"
)

// adminCustomCSS — убрать лишнее, удобный и стильный интерфейс.
const adminCustomCSS = `
<style>
/* === Масштаб и база === */
html { zoom: 1.3; }
body { font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, "Helvetica Neue", Arial, sans-serif; }

/* === Убрать лишнее === */
.main-footer .pull-right, .main-footer a[href*="goadmin"], .main-footer .hidden-xs { display: none !important; }
/* Убрать картинку (pull-left) из шапки рядом с гамбургер-меню */
.main-header .navbar .pull-left img,
.main-header .navbar .pull-left .brand-image,
.main-header .navbar .pull-left > img { display: none !important; }
.main-footer { padding: 10px 15px; font-size: 12px; color: #999; }
.content-header .breadcrumb { display: none !important; }
.content-header h1 { margin: 0; font-size: 22px; font-weight: 600; }
.box .box-header .box-title { font-size: 16px; font-weight: 600; }
.breadcrumb { background: transparent !important; padding: 0 !important; }

/* === Логотип в сайдбаре — показывать полностью, не обрезать === */
.main-sidebar .logo, .main-sidebar .logo-mini { overflow: visible !important; height: auto !important; min-height: 50px; }
.main-sidebar .logo a, .main-sidebar .logo-mini a { white-space: nowrap; overflow: visible; padding: 12px 15px; font-size: 16px; display: block; }
.skin-black .main-sidebar .logo, .skin-black .main-sidebar .logo-mini { background: transparent; border-bottom: 1px solid #333; }

/* === Боковое меню — чище и стильнее === */
.main-sidebar { padding-top: 0; }
.main-sidebar .sidebar-menu li a { border-left: 3px solid transparent; padding: 12px 15px; transition: all .2s; }
.main-sidebar .sidebar-menu li.active > a,
.main-sidebar .sidebar-menu li a:hover { border-left-color: #3498db; background: rgba(52,152,219,.1) !important; }
.main-sidebar .sidebar-menu > li > a > .fa { width: 22px; margin-right: 8px; }
.skin-black .main-sidebar .sidebar-menu li.active > a,
.skin-black .main-sidebar .sidebar-menu li a:hover { background: rgba(52,152,219,.15) !important; border-left-color: #5dade2; }

/* === Карточки и блоки === */
.content-wrapper .box { border-radius: 8px; box-shadow: 0 1px 3px rgba(0,0,0,.08); border: 1px solid #eee; }
.content-wrapper .box .box-header { border-radius: 8px 8px 0 0; padding: 14px 16px; background: #fafafa; border-bottom: 1px solid #eee; }
.content-wrapper .box .box-body { padding: 16px; }
.skin-black .content-wrapper .box { border-color: #333; }
.skin-black .content-wrapper .box .box-header { background: #1a2226; border-color: #333; }

/* === Кнопки === */
.btn { border-radius: 6px; font-weight: 500; transition: all .2s; }
.btn-primary { background: #3498db; border-color: #2980b9; }
.btn-primary:hover { background: #2980b9; border-color: #21618c; }
.btn-success { background: #27ae60; border-color: #1e8449; }
.btn-success:hover { background: #1e8449; }
.content-wrapper .table-responsive { border-radius: 6px; overflow: hidden; }

/* === Таблица и Operation — два отдельных блока === */
.content-wrapper .table-responsive { overflow-x: auto; }
.content-wrapper .table { table-layout: auto; min-width: 1200px; margin: 0; background: #fff; }
.content-wrapper .table th,
.content-wrapper .table td { white-space: nowrap; vertical-align: middle !important; padding: 10px 12px !important; }
.content-wrapper .table thead th { background: #f8f9fa !important; font-weight: 600; color: #333; border-bottom: 2px solid #dee2e6; }
.content-wrapper .table tbody tr:nth-child(even) td:not(:last-child) { background: #fafafa; }
.content-wrapper .table td:first-child,
.content-wrapper .table th:first-child { min-width: 280px; white-space: normal; word-break: break-all; }
.content-wrapper .table th { min-width: 80px; }
.content-wrapper .table th:nth-last-child(2),
.content-wrapper .table td:nth-last-child(2) { min-width: 100px; }

/* Колонка Operation — узкая, одна кнопка ⋮; действия в модальном окне (admin-custom.js) */
.content-wrapper .table th:last-child,
.content-wrapper .table td:last-child {
  position: sticky; right: 0; min-width: 52px; max-width: 52px; width: 52px;
  text-align: center; padding: 6px 4px !important; z-index: 2;
  background: #f0f4f8 !important; border-left: 1px solid #dee2e6;
}
.content-wrapper .table thead th:last-child {
  background: #e9ecef !important; color: #333; font-weight: 600; font-size: 12px;
}
.admin-actions-trigger {
  display: inline-block; width: 32px; height: 32px; padding: 0;
  border: 1px solid #3498db; background: #3498db; color: #fff;
  border-radius: 6px; font-size: 18px; line-height: 1; cursor: pointer;
  font-weight: bold; vertical-align: middle;
}
.admin-actions-trigger:hover { background: #2980b9; border-color: #2980b9; color: #fff; }
.admin-actions-hidden { display: none !important; }

/* Модальное окно «Действия» */
.admin-actions-modal { display: none; position: fixed; inset: 0; z-index: 9999; }
.admin-actions-modal.admin-actions-open { display: block; }
.admin-actions-overlay { position: absolute; inset: 0; background: rgba(0,0,0,.4); }
.admin-actions-box {
  position: absolute; left: 50%; top: 50%; transform: translate(-50%,-50%);
  background: #fff; border-radius: 10px; box-shadow: 0 10px 40px rgba(0,0,0,.2);
  padding: 20px; min-width: 200px;
}
.admin-actions-title { font-size: 14px; font-weight: 600; margin-bottom: 12px; color: #333; }
.admin-actions-links { display: flex; flex-direction: column; gap: 8px; margin-bottom: 14px; }
.admin-actions-links a.admin-actions-btn {
  display: block; padding: 10px 14px; background: #3498db; color: #fff !important;
  border-radius: 6px; text-align: center; text-decoration: none; font-size: 13px;
}
.admin-actions-links a.admin-actions-btn:hover { background: #2980b9; color: #fff !important; }
.admin-actions-close {
  display: block; width: 100%; padding: 8px; border: 1px solid #ddd;
  background: #f8f9fa; border-radius: 6px; cursor: pointer; font-size: 13px;
}
.admin-actions-close:hover { background: #e9ecef; }

/* Тёмная тема */
.skin-black .content-wrapper .table thead th { background: #2c3b41 !important; color: #ecf0f1; }
.skin-black .content-wrapper .table tbody tr:nth-child(even) td:not(:last-child) { background: #252d33; }
.skin-black .content-wrapper .table th:last-child,
.skin-black .content-wrapper .table td:last-child { background: #252d33 !important; border-left-color: #333; }
.skin-black .content-wrapper .table thead th:last-child { background: #1a2226 !important; color: #ecf0f1; }
.skin-black .admin-actions-box { background: #222d32; }
.skin-black .admin-actions-title { color: #ecf0f1; }
.skin-black .admin-actions-close { background: #1a2226; border-color: #333; color: #ecf0f1; }
.skin-black .admin-actions-close:hover { background: #252d33; }

/* === Формы и фильтры === */
.form-control { border-radius: 6px; border: 1px solid #ddd; }
.form-control:focus { border-color: #3498db; box-shadow: 0 0 0 3px rgba(52,152,219,.15); }

/* === Карточки на главной === */
a.admin-dash-card {
  display: block; text-decoration: none; color: inherit; background: #fff;
  border: 1px solid #e8e8e8; border-radius: 10px; padding: 24px; margin-bottom: 20px;
  box-shadow: 0 2px 8px rgba(0,0,0,.06); transition: all .25s;
}
a.admin-dash-card:hover { border-color: #3498db; box-shadow: 0 4px 16px rgba(52,152,219,.2); transform: translateY(-2px); color: inherit; }
.admin-dash-card-icon { width: 48px; height: 48px; border-radius: 10px; background: #e8f4fc; color: #3498db; display: flex; align-items: center; justify-content: center; margin-bottom: 14px; font-size: 22px; }
a.admin-dash-card:hover .admin-dash-card-icon { background: #3498db; color: #fff; }
.admin-dash-card-title { font-size: 16px; font-weight: 600; color: #222; margin-bottom: 4px; }
.admin-dash-card-desc { font-size: 13px; color: #666; }
.skin-black a.admin-dash-card { background: #222d32; border-color: #333; }
.skin-black a.admin-dash-card:hover { border-color: #3498db; }
.skin-black .admin-dash-card-title { color: #ecf0f1; }
.skin-black .admin-dash-card-desc { color: #95a5a6; }
.skin-black .admin-dash-card-icon { background: #1e3a4a; color: #5dade2; }
</style>
`

// adminCustomJS — в колонке Operation одна кнопка; по клику модальное окно с Edit/delete/view.
const adminCustomJS = "(function(){function ready(fn){if(document.readyState!=='loading')fn();else document.addEventListener('DOMContentLoaded',fn);}ready(function(){var brand=document.querySelector('.navbar-brand');if(brand)brand.innerHTML=brand.innerHTML.replace(/GoAdmin/g,'Sarbon Admin');var tables=document.querySelectorAll('.content-wrapper .table');if(!tables.length)return;var modal=document.createElement('div');modal.className='admin-actions-modal';modal.innerHTML='<div class=\"admin-actions-overlay\"></div><div class=\"admin-actions-box\"><div class=\"admin-actions-title\">Действия</div><div class=\"admin-actions-links\"></div><button type=\"button\" class=\"admin-actions-close\">Закрыть</button></div>';document.body.appendChild(modal);var linksContainer=modal.querySelector('.admin-actions-links');function closeModal(){modal.classList.remove('admin-actions-open');}modal.querySelector('.admin-actions-overlay').addEventListener('click',closeModal);modal.querySelector('.admin-actions-close').addEventListener('click',closeModal);tables.forEach(function(table){var cells=table.querySelectorAll('tbody tr td:last-child');cells.forEach(function(cell){if(cell.querySelector('.admin-actions-cell'))return;var as=cell.querySelectorAll('a');if(!as.length)return;var wrap=document.createElement('div');wrap.className='admin-actions-cell';var btn=document.createElement('button');btn.type='button';btn.className='admin-actions-trigger';btn.textContent='\u22EE';btn.title='Действия';var hidden=document.createElement('div');hidden.className='admin-actions-hidden';hidden.style.display='none';for(var i=0;i<as.length;i++)hidden.appendChild(as[i].cloneNode(true));wrap.appendChild(btn);wrap.appendChild(hidden);cell.innerHTML='';cell.appendChild(wrap);btn.addEventListener('click',function(){linksContainer.innerHTML='';var list=hidden.querySelectorAll('a');for(var j=0;j<list.length;j++){var link=list[j].cloneNode(true);link.className='admin-actions-btn';linksContainer.appendChild(link);}modal.classList.add('admin-actions-open');});});});});})();"

// Mount mounts the GoAdmin panel on the Gin router.
// databaseURL must be a Postgres URL (e.g. postgres://user:pass@host:5432/dbname?sslmode=disable).
// Panel will be at /admin (login: admin / admin by default).
func Mount(r *gin.Engine, databaseURL string) error {
	dbCfg := parseDatabaseURL(databaseURL)
	uploadsPath := "./uploads"
	_ = os.MkdirAll(uploadsPath, 0755)

	cfg := &config.Config{
		Env: config.EnvLocal,
		Databases: config.DatabaseList{
			"default": dbCfg,
		},
		UrlPrefix: "admin",
		Store: config.Store{
			Path:   uploadsPath,
			Prefix: "uploads",
		},
		Language:           language.EN,
		IndexUrl:           "/",
		Debug:              true,
		AccessAssetsLogOff: true,
		ColorScheme:        adminlte.ColorschemeSkinBlack,
		Title:              "Sarbon Admin",
		LoginTitle:         "Sarbon Admin",
		Logo:               template.HTML("<b>Sarbon</b> Admin"),
		MiniLogo:           template.HTML("<b>S</b>A"),
		// Стили подключаем по URL
		CustomHeadHtml: template.HTML(`<link rel="stylesheet" href="/admin-custom.css">`),
	}

	eng := engine.Default()
	generators := tableGenerators()
	if err := eng.AddConfig(cfg).
		AddGenerators(generators).
		AddDisplayFilterXssJsFilter().
		Use(r); err != nil {
		return err
	}

	// Register index page (GET /admin) — without this, /admin returns 404
	eng.HTML("GET", "/admin", DashboardContent)

	// Раздача кастомного CSS по URL (тема может не выводить CustomHeadHtml)
	r.GET("/admin-custom.css", func(c *gin.Context) {
		css := strings.TrimPrefix(adminCustomCSS, "<style>")
		css = strings.TrimSuffix(css, "</style>")
		css = strings.TrimSpace(css)
		c.Header("Content-Type", "text/css; charset=utf-8")
		c.String(200, css)
	})
	r.GET("/admin-custom.js", func(c *gin.Context) {
		c.Header("Content-Type", "application/javascript; charset=utf-8")
		c.String(200, adminCustomJS)
	})

	r.Static("/uploads", filepath.Clean(uploadsPath))
	return nil
}

func parseDatabaseURL(raw string) config.Database {
	raw = strings.TrimSpace(raw)
	// Support pgx-style URL for consistency with app
	if strings.HasPrefix(raw, "pgx5://") {
		raw = "postgres://" + strings.TrimPrefix(raw, "pgx5://")
	}
	if strings.HasPrefix(raw, "pgx://") {
		raw = "postgres://" + strings.TrimPrefix(raw, "pgx://")
	}

	u, err := url.Parse(raw)
	if err != nil {
		return config.Database{
			Driver:          config.DriverPostgresql,
			Dsn:             raw,
			MaxIdleConns:    10,
			MaxOpenConns:    100,
			ConnMaxLifetime: time.Hour,
		}
	}

	password, _ := u.User.Password()
	port := u.Port()
	if port == "" {
		port = "5432"
	}
	dbname := strings.TrimPrefix(u.Path, "/")
	if dbname == "" {
		dbname = "postgres"
	}

	return config.Database{
		Host:            u.Hostname(),
		Port:            port,
		User:            u.User.Username(),
		Pwd:             password,
		Name:            dbname,
		Driver:          config.DriverPostgresql,
		MaxIdleConns:    10,
		MaxOpenConns:    100,
		ConnMaxLifetime: time.Hour,
	}
}

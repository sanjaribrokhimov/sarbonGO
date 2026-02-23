-- GoAdmin panel tables (PostgreSQL). Default login: admin / admin.

CREATE TABLE IF NOT EXISTS goadmin_menu (
  id SERIAL PRIMARY KEY,
  parent_id INTEGER NOT NULL DEFAULT 0,
  type SMALLINT NOT NULL DEFAULT 0,
  "order" INTEGER NOT NULL DEFAULT 0,
  title VARCHAR(50) NOT NULL,
  icon VARCHAR(50) NOT NULL,
  uri VARCHAR(3000) NOT NULL DEFAULT '',
  header VARCHAR(150),
  plugin_name VARCHAR(150) NOT NULL DEFAULT '',
  uuid VARCHAR(150),
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS goadmin_operation_log (
  id SERIAL PRIMARY KEY,
  user_id INTEGER NOT NULL,
  path VARCHAR(255) NOT NULL,
  method VARCHAR(10) NOT NULL,
  ip VARCHAR(15) NOT NULL,
  input TEXT NOT NULL,
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX IF NOT EXISTS goadmin_operation_log_user_id_idx ON goadmin_operation_log (user_id);

CREATE TABLE IF NOT EXISTS goadmin_site (
  id SERIAL PRIMARY KEY,
  key VARCHAR(100),
  value TEXT,
  description VARCHAR(3000),
  state SMALLINT NOT NULL DEFAULT 0,
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS goadmin_permissions (
  id SERIAL PRIMARY KEY,
  name VARCHAR(50) NOT NULL,
  slug VARCHAR(50) NOT NULL,
  http_method VARCHAR(255),
  http_path TEXT NOT NULL,
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  CONSTRAINT goadmin_permissions_name_unique UNIQUE (name)
);

CREATE TABLE IF NOT EXISTS goadmin_roles (
  id SERIAL PRIMARY KEY,
  name VARCHAR(50) NOT NULL,
  slug VARCHAR(50) NOT NULL,
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  CONSTRAINT goadmin_roles_name_unique UNIQUE (name)
);

CREATE TABLE IF NOT EXISTS goadmin_users (
  id SERIAL PRIMARY KEY,
  username VARCHAR(100) NOT NULL,
  password VARCHAR(100) NOT NULL DEFAULT '',
  name VARCHAR(100) NOT NULL,
  avatar VARCHAR(255),
  remember_token VARCHAR(100),
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  CONSTRAINT goadmin_users_username_unique UNIQUE (username)
);

CREATE TABLE IF NOT EXISTS goadmin_session (
  id SERIAL PRIMARY KEY,
  sid VARCHAR(50) NOT NULL DEFAULT '',
  values VARCHAR(3000) NOT NULL,
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS goadmin_role_menu (
  role_id INTEGER NOT NULL,
  menu_id INTEGER NOT NULL,
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  CONSTRAINT goadmin_role_menu_unique UNIQUE (role_id, menu_id)
);
CREATE INDEX IF NOT EXISTS goadmin_role_menu_role_id_menu_id_idx ON goadmin_role_menu (role_id, menu_id);

CREATE TABLE IF NOT EXISTS goadmin_role_permissions (
  role_id INTEGER NOT NULL,
  permission_id INTEGER NOT NULL,
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  CONSTRAINT goadmin_role_permissions_unique UNIQUE (role_id, permission_id)
);

CREATE TABLE IF NOT EXISTS goadmin_role_users (
  role_id INTEGER NOT NULL,
  user_id INTEGER NOT NULL,
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  CONSTRAINT goadmin_role_users_unique UNIQUE (role_id, user_id)
);

CREATE TABLE IF NOT EXISTS goadmin_user_permissions (
  user_id INTEGER NOT NULL,
  permission_id INTEGER NOT NULL,
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  CONSTRAINT goadmin_user_permissions_unique UNIQUE (user_id, permission_id)
);

-- Seed: permissions
INSERT INTO goadmin_permissions (id, name, slug, http_method, http_path, created_at, updated_at)
VALUES
  (1, 'All permission', '*', '', '*', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
  (2, 'Dashboard', 'dashboard', 'GET,PUT,POST,DELETE', '/', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
ON CONFLICT (name) DO NOTHING;

-- Seed: roles
INSERT INTO goadmin_roles (id, name, slug, created_at, updated_at)
VALUES
  (1, 'Administrator', 'administrator', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
  (2, 'Operator', 'operator', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
ON CONFLICT (name) DO NOTHING;

-- Seed: default admin user (password: admin)
INSERT INTO goadmin_users (id, username, password, name, avatar, remember_token, created_at, updated_at)
VALUES
  (1, 'admin', '$2a$10$U3F/NSaf2kaVbyXTBp7ppOn0jZFyRqXRnYXB.AMioCjXl3Ciaj4oy', 'admin', '', NULL, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
ON CONFLICT (username) DO NOTHING;

-- Seed: role_users (admin has role 1)
INSERT INTO goadmin_role_users (role_id, user_id, created_at, updated_at)
VALUES (1, 1, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
ON CONFLICT (role_id, user_id) DO NOTHING;

-- Seed: role_permissions
INSERT INTO goadmin_role_permissions (role_id, permission_id, created_at, updated_at)
VALUES
  (1, 1, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
  (1, 2, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
  (2, 2, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
ON CONFLICT (role_id, permission_id) DO NOTHING;

-- Seed: user_permissions
INSERT INTO goadmin_user_permissions (user_id, permission_id, created_at, updated_at)
VALUES
  (1, 1, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
  (2, 2, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
ON CONFLICT (user_id, permission_id) DO NOTHING;

-- Seed: menu (Dashboard + Admin section; add Companies and Admins)
INSERT INTO goadmin_menu (id, parent_id, type, "order", title, icon, uri, plugin_name, header, created_at, updated_at)
VALUES
  (1, 0, 1, 2, 'Admin', 'fa-tasks', '', '', NULL, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
  (2, 1, 1, 2, 'Users', 'fa-users', '/info/goadmin_user', '', NULL, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
  (3, 1, 1, 3, 'Roles', 'fa-user', '/info/roles', '', NULL, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
  (4, 1, 1, 4, 'Permission', 'fa-ban', '/info/permission', '', NULL, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
  (5, 1, 1, 5, 'Menu', 'fa-bars', '/menu', '', NULL, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
  (6, 1, 1, 6, 'Operation log', 'fa-history', '/info/op', '', NULL, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
  (7, 0, 1, 1, 'Dashboard', 'fa-bar-chart', '/', '', NULL, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
  (8, 0, 1, 3, 'Companies', 'fa-building', '/info/companies', '', NULL, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
  (9, 0, 1, 4, 'App Admins', 'fa-user-secret', '/info/admins', '', NULL, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
ON CONFLICT (id) DO NOTHING;

INSERT INTO goadmin_role_menu (role_id, menu_id, created_at, updated_at)
VALUES
  (1, 1, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
  (1, 7, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
  (2, 7, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
  (1, 8, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
  (2, 8, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
  (1, 9, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
ON CONFLICT (role_id, menu_id) DO NOTHING;

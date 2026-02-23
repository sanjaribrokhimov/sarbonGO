-- Simplify GoAdmin menu: only table management (no Users, Roles, Permission, Menu, Operation log).

DELETE FROM goadmin_role_menu;

DELETE FROM goadmin_menu WHERE id IN (1, 2, 3, 4, 5, 6, 7, 8, 9);

INSERT INTO goadmin_menu (id, parent_id, type, "order", title, icon, uri, plugin_name, header, created_at, updated_at)
VALUES
  (7,  0, 1, 1, 'Dashboard',     'fa-dashboard', '/', '', NULL, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
  (8,  0, 1, 2, 'Companies',     'fa-building',   '/info/companies', '', NULL, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
  (9,  0, 1, 3, 'Operator Admins', 'fa-user-plus', '/info/admins', '', NULL, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
  (10, 0, 1, 4, 'Drivers',       'fa-truck',     '/info/drivers', '', NULL, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
  (11, 0, 1, 5, 'Freelance Dispatchers', 'fa-users', '/info/freelance_dispatchers', '', NULL, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
ON CONFLICT (id) DO NOTHING;

-- Administrator (role 1): all menus
INSERT INTO goadmin_role_menu (role_id, menu_id, created_at, updated_at)
SELECT 1, id, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP FROM goadmin_menu WHERE id IN (7, 8, 9, 10, 11)
ON CONFLICT (role_id, menu_id) DO NOTHING;

-- Operator (role 2): Companies + Operator Admins (can create companies and manage operator admins)
INSERT INTO goadmin_role_menu (role_id, menu_id, created_at, updated_at)
VALUES
  (2, 7, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
  (2, 8, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
  (2, 9, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
ON CONFLICT (role_id, menu_id) DO NOTHING;

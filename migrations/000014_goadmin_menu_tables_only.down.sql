-- Restore previous menu (same as 000013 seed).

DELETE FROM goadmin_role_menu;
DELETE FROM goadmin_menu WHERE id IN (7, 8, 9, 10, 11);

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
  (9, 0, 1, 4, 'App Admins', 'fa-user-secret', '/info/admins', '', NULL, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP);

INSERT INTO goadmin_role_menu (role_id, menu_id, created_at, updated_at)
VALUES
  (1, 1, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
  (1, 7, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
  (2, 7, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
  (1, 8, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
  (2, 8, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
  (1, 9, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP);

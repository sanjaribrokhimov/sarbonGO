-- Add Cargo, Route points, Payments, Offers to GoAdmin menu.

INSERT INTO goadmin_menu (id, parent_id, type, "order", title, icon, uri, plugin_name, header, created_at, updated_at)
VALUES
  (12, 0, 1, 6, 'Cargo',           'fa-cube',     '/info/cargo', '', NULL, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
  (13, 0, 1, 7, 'Route Points',   'fa-map-marker', '/info/route_points', '', NULL, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
  (14, 0, 1, 8, 'Payments',       'fa-money',    '/info/payments', '', NULL, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
  (15, 0, 1, 9, 'Offers',         'fa-handshake-o', '/info/offers', '', NULL, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
ON CONFLICT (id) DO NOTHING;

INSERT INTO goadmin_role_menu (role_id, menu_id, created_at, updated_at)
SELECT 1, id, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP FROM goadmin_menu WHERE id IN (12, 13, 14, 15)
ON CONFLICT (role_id, menu_id) DO NOTHING;

INSERT INTO goadmin_role_menu (role_id, menu_id, created_at, updated_at)
SELECT 2, id, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP FROM goadmin_menu WHERE id IN (12, 13, 14, 15)
ON CONFLICT (role_id, menu_id) DO NOTHING;

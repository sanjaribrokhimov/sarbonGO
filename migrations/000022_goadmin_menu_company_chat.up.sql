-- GoAdmin menu: Company TZ (app_users, app_roles, user_company_roles, invitations, audit_log) and Chat.

INSERT INTO goadmin_menu (id, parent_id, type, "order", title, icon, uri, plugin_name, header, created_at, updated_at)
VALUES
  (16, 0, 1, 10, 'App Users',       'fa-user',        '/info/app_users', '', NULL, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
  (17, 0, 1, 11, 'App Roles',       'fa-user-circle', '/info/app_roles', '', NULL, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
  (18, 0, 1, 12, 'User Company Roles', 'fa-link',      '/info/user_company_roles', '', NULL, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
  (19, 0, 1, 13, 'Invitations',     'fa-envelope',    '/info/invitations', '', NULL, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
  (20, 0, 1, 14, 'Audit Log',       'fa-history',     '/info/audit_log', '', NULL, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
  (21, 0, 1, 15, 'Chat Conversations', 'fa-comments',  '/info/chat_conversations', '', NULL, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
  (22, 0, 1, 16, 'Chat Messages',   'fa-comment',     '/info/chat_messages', '', NULL, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
ON CONFLICT (id) DO NOTHING;

-- Role 1 (Administrator): full access to new menus
INSERT INTO goadmin_role_menu (role_id, menu_id, created_at, updated_at)
SELECT 1, id, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP FROM goadmin_menu WHERE id IN (16, 17, 18, 19, 20, 21, 22)
ON CONFLICT (role_id, menu_id) DO NOTHING;

-- Role 2 (Operator): access to Company TZ and Chat
INSERT INTO goadmin_role_menu (role_id, menu_id, created_at, updated_at)
SELECT 2, id, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP FROM goadmin_menu WHERE id IN (16, 17, 18, 19, 20, 21, 22)
ON CONFLICT (role_id, menu_id) DO NOTHING;

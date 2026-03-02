-- Company module by TZ 5.2: users, roles, user_company_roles, invitations, audit_log; companies.owner_id + type.

CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- 1) users (app users for auth/register and company ownership)
CREATE TABLE IF NOT EXISTS app_users (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    email VARCHAR(255) UNIQUE NOT NULL,
    phone VARCHAR(20) UNIQUE,
    password_hash VARCHAR(255) NOT NULL,
    first_name VARCHAR(100),
    last_name VARCHAR(100),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX IF NOT EXISTS idx_app_users_email ON app_users (email);
CREATE INDEX IF NOT EXISTS idx_app_users_phone ON app_users (phone) WHERE phone IS NOT NULL;

-- 2) roles (reference)
CREATE TABLE IF NOT EXISTS app_roles (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name VARCHAR(50) UNIQUE NOT NULL,
    description TEXT
);

-- 3) companies: add owner_id and type for TZ (keep existing columns for backward compat)
ALTER TABLE companies
  ADD COLUMN IF NOT EXISTS owner_id UUID NULL,
  ADD COLUMN IF NOT EXISTS company_type VARCHAR(20) NULL,
  ADD COLUMN IF NOT EXISTS auto_approve_limit DECIMAL(10, 2) NULL;

DO $$
BEGIN
  IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'companies_type_check') THEN
    ALTER TABLE companies ADD CONSTRAINT companies_type_check
    CHECK (company_type IS NULL OR company_type IN ('Shipper', 'Broker', 'Fleet', 'OwnerOperator'));
  END IF;
END$$;

-- FK owner_id -> app_users (add only if app_users exists)
DO $$
BEGIN
  IF to_regclass('public.app_users') IS NOT NULL THEN
    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'fk_companies_owner_app_users') THEN
      ALTER TABLE companies
        ADD CONSTRAINT fk_companies_owner_app_users
        FOREIGN KEY (owner_id) REFERENCES app_users(id);
    END IF;
  END IF;
END$$;

-- 4) user_company_roles
CREATE TABLE IF NOT EXISTS user_company_roles (
    user_id UUID NOT NULL REFERENCES app_users(id) ON DELETE CASCADE,
    company_id UUID NOT NULL REFERENCES companies(id) ON DELETE CASCADE,
    role_id UUID NOT NULL REFERENCES app_roles(id),
    assigned_by UUID REFERENCES app_users(id),
    assigned_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (user_id, company_id, role_id)
);
CREATE INDEX IF NOT EXISTS idx_user_company_roles_company ON user_company_roles (company_id);
CREATE INDEX IF NOT EXISTS idx_user_company_roles_user ON user_company_roles (user_id);

-- 5) invitations
CREATE TABLE IF NOT EXISTS invitations (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    token VARCHAR(64) UNIQUE NOT NULL,
    company_id UUID NOT NULL REFERENCES companies(id) ON DELETE CASCADE,
    role_id UUID NOT NULL REFERENCES app_roles(id),
    email VARCHAR(255) NOT NULL,
    invited_by UUID NOT NULL REFERENCES app_users(id),
    expires_at TIMESTAMP NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
CREATE UNIQUE INDEX IF NOT EXISTS idx_invitations_token ON invitations (token);
CREATE INDEX IF NOT EXISTS idx_invitations_email ON invitations (email);
CREATE INDEX IF NOT EXISTS idx_invitations_company ON invitations (company_id);

-- 6) audit_log
CREATE TABLE IF NOT EXISTS audit_log (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID REFERENCES app_users(id),
    company_id UUID REFERENCES companies(id),
    action VARCHAR(50) NOT NULL,
    entity_type VARCHAR(50) NOT NULL,
    entity_id UUID NOT NULL,
    old_data JSONB,
    new_data JSONB,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX IF NOT EXISTS idx_audit_log_company ON audit_log (company_id);
CREATE INDEX IF NOT EXISTS idx_audit_log_created ON audit_log (created_at DESC);

-- 7) Seed roles (idempotent by name)
INSERT INTO app_roles (name, description) VALUES
('Owner', 'Владелец компании, полный доступ ко всем ресурсам компании, включая удаление компании. Не может быть приглашён в другую компанию как Owner.'),
('CEO', 'Директор, управление компанией, назначение ролей (кроме Owner), просмотр всей аналитики, подтверждение крупных ставок.'),
('TopDispatcher', 'Старший диспетчер, управление диспетчерами, назначение/снятие, просмотр всех ставок.'),
('Dispatcher', 'Диспетчер, просмотр и назначение ставок в рамках лимита автоодобрения, не может публиковать грузы.'),
('TopManager', 'Старший менеджер, управление менеджерами, публикация грузов, подтверждение ставок, управление пользователями.'),
('Manager', 'Менеджер, публикация грузов, подтверждение ставок, управление назначенными грузами.'),
('Carrier', 'Перевозчик, создание ставок на грузы, назначение водителей на рейсы, просмотр своих рейсов.'),
('Driver', 'Водитель, доступ через мобильное приложение, просмотр назначенных рейсов, изменение статусов рейса, чат.')
ON CONFLICT (name) DO NOTHING;

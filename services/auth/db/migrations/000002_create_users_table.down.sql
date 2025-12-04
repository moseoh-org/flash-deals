-- Auth Service: users 테이블 삭제
DROP TRIGGER IF EXISTS update_users_updated_at ON auth.users;
DROP FUNCTION IF EXISTS auth.update_updated_at_column();
DROP INDEX IF EXISTS auth.idx_users_email;
DROP TABLE IF EXISTS auth.users;

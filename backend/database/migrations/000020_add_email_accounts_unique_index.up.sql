-- 为 email_accounts 增加同用户同 provider + email 的唯一性约束。
-- 若本 migration 因重复数据失败，请先执行以下查询定位历史重复：
-- SELECT user_id, email, provider, COUNT(*) AS cnt
-- FROM email_accounts
-- WHERE deleted_at IS NULL
-- GROUP BY user_id, email, provider
-- HAVING COUNT(*) > 1;
CREATE UNIQUE INDEX IF NOT EXISTS idx_email_accounts_user_email_provider_unique
ON email_accounts(user_id, email, provider)
WHERE deleted_at IS NULL;

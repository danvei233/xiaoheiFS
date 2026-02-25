START TRANSACTION;

INSERT INTO settings (`key`, `value_json`, `updated_at`) VALUES
('auth_register_sms_template_id', '', NOW()),
('sms_default_template_id', '', NOW()),
('sms_enabled', 'true', NOW()),
('sms_instance_id', 'default', NOW()),
('sms_plugin_id', 'duanxinbao', NOW()),
('sms_provider_template_id', '', NOW()),
('sms_templates_json', '[{"id":1,"name":"register_verify_code","content":"【XXX】您正在注册XXX平台账号，验证码是：{{code}}，3分钟内有效，请及时输入。","enabled":true},{"id":2,"name":"login_ip_change_alert","content":"【XXX】登录提醒：您的账号于 {{time}} 在 {{city}} 发生登录（IP：{{ip}}）。如为本人操作，请忽略本消息；如非本人操作，请立即修改密码并开启二次验证，确保账号安全。","enabled":true},{"id":3,"name":"password_reset_verify_code","content":"【XXX】您好，您在XXX平台（APP）的账号正在进行找回密码操作，切勿将验证码泄露于他人，10分钟内有效。验证码：{{code}}。","enabled":true},{"id":4,"name":"phone_bind_verify_code","content":"【XXX】手机绑定验证码：{{code}}，感谢您的支持！如非本人操作，请忽略本短信。","enabled":true},{"id":5,"name":"phone_change_alert_old_contact","content":"【XXX】安全提醒：您的账号手机号已于 {{time}} 从 {{old_phone}} 修改为 {{new_phone}}。如非本人操作，请立即修改密码并联系管理员。","enabled":true}]', NOW())
ON DUPLICATE KEY UPDATE
`value_json` = VALUES(`value_json`),
`updated_at` = VALUES(`updated_at`);

COMMIT;

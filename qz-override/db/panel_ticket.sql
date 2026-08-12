-- ============================================================================
-- 持久化迁移/安装契约：短时单次 ticket 表（panel_ticket）
--
-- 由 `app/common/service/TicketService.php` 的 DbTicketStore 使用。面板/VNC
-- 跳转是两次独立 HTTP 请求（/api/cloud/panel、/api/cloud/vnc_view 签发 →
-- control/ecs/login、control/ecs/vnc 消费），ticket 必须跨请求持久化，因此
-- 生产环境必须执行本文件创建表。缺表时签发/消费会报错（fail-closed），不会
-- 退化成内存存储（防多实例丢票/跨实例重放）。
--
-- 安装：
--   mysql -u<user> -p <db> < qz-override/db/panel_ticket.sql
-- 或经 ThinkPHP migrate（`php think migrate:run`，把本文件注册为迁移）。
--
-- 安全属性由应用层保证（见 TicketService）：
--   * ticket 为 random_bytes(32) hex，不可预测/不可枚举；
--   * TTL 60s，`expires_at` 校验过期即拒；
--   * 单次消费：`used 0 -> 1` 由 UPDATE ... WHERE used=0 原子完成，并发重放
--     只有一个请求影响 1 行（并发单赢家）；
--   * 绑定：kind/host/caller 在签发时写入，消费时从行取出并交叉核对；
--   * 速率限制：同一 (kind,host,caller) 在窗口内最多签发 MAX_PER_WINDOW 张。
-- ============================================================================

CREATE TABLE IF NOT EXISTS `panel_ticket` (
    `ticket`     VARCHAR(64)  NOT NULL COMMENT 'random_bytes(32) hex；单次 ticket，主键',
    `kind`       VARCHAR(16)  NOT NULL COMMENT "'panel' | 'vnc'；消费时必须匹配",
    `host`       VARCHAR(64)  NOT NULL COMMENT '绑定的主机（panel=host_name / vnc=host_id）',
    `caller`     VARCHAR(64)  NOT NULL COMMENT '调用方凭据 id（签发时校验所有权的租户标识）',
    `expires_at` INT          NOT NULL COMMENT 'unix 时间戳；超过即拒绝消费',
    `used`       TINYINT(1)   NOT NULL DEFAULT 0 COMMENT '0 未消费 / 1 已消费（单次）',
    `created_at` INT          NOT NULL COMMENT 'unix 时间戳；用于速率限制窗口',
    PRIMARY KEY (`ticket`),
    KEY `idx_kind_host_caller` (`kind`, `host`, `caller`),
    KEY `idx_expires_at` (`expires_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='面板/VNC 短时单次跳转 ticket';

-- 周期清理过期行（应用层每次 issue 也会顺手 deleteExpired；此处为运维兜底）。
-- 可放入 cron：每天删除已过期一天以上的 ticket。
-- DELETE FROM `panel_ticket` WHERE `expires_at` < UNIX_TIMESTAMP() - 86400;

-- order_auth：续期/升级「已支付订单授权」安装契约（Issue #51 P2 残余复核）。
--
-- 用途：renew / update / elastic_update 不能仅凭 order 作用域 + 未来时间窗
-- 免费续期/换高配产品。支付集成在订单支付成功后 INSERT 一条 status='paid' 的
-- 授权记录，目标端点在另一请求用 OrderAuthorizationService::consume 原子消费
-- （单次/幂等/越权绑定）后才允许变更 end_time/product/resources。
--
-- 部署：生产必须执行本文件建表；缺表时消费 fail-closed（不退化内存存储）。
CREATE TABLE IF NOT EXISTS `order_auth` (
  `order_id`    VARCHAR(64)  NOT NULL COMMENT '已支付订单号/授权号（唯一）',
  `kind`        VARCHAR(32)  NOT NULL COMMENT '用途：renew / update / elastic_update',
  `user_id`     INT          NOT NULL COMMENT '订单归属用户（越权绑定）',
  `host_id`     VARCHAR(64)  NOT NULL COMMENT '授权目标主机（跨主机拒绝）',
  `status`      VARCHAR(16)  NOT NULL DEFAULT 'paid' COMMENT 'paid=已支付待消费 / used=已消费 / revoked=撤销',
  `expires_at`  INT          NOT NULL COMMENT '授权过期时间（unix 秒；默认签发后 TTL=3600）',
  `consumed_at` INT          NULL COMMENT '消费时间（原子 markConsumed 写入）',
  `created_at`  INT          NOT NULL COMMENT '创建时间',
  PRIMARY KEY (`order_id`),
  KEY `idx_owner` (`user_id`, `host_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='续期/升级已支付订单授权（单次消费）';

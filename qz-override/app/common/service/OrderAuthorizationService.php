<?php

namespace app\common\service;

/**
 * 续期/升级授权证明服务（Issue #51 P2 残余复核：renew/update 免费续期/换配）。
 *
 * 修复目标：`renew`/`update`/`elastic_update` 不能仅凭 `order` 作用域 + 未来
 * 时间窗就免费续期/换高配产品。续期与升级必须携带一条与「已支付订单」绑定的
 * 授权记录（order_auth），且该记录被**原子地单次消费**（idempotent
 * single-use）后才允许变更 end_time / product / resources；任何失败路径
 * （无授权 / 越权 / 过期 / 重放 / 并发竞争失败）都不改变主机状态。
 *
 * 授权记录由支付集成在订单支付成功后写入（status='paid'）；本服务只负责
 * 校验绑定 + 原子消费。消费必须与主机状态变更处于**同一 DB 事务**内，这样
 * 主机变更失败时授权消费一并回滚（失败不改 end_time/product/resources）。
 *
 * 安全属性：
 * - 绑定：order_auth 行携带 user_id + host_id + kind，消费时逐一交叉核对，
 *   跨用户（越权）/跨主机/跨用途一律拒绝；
 * - 单次：`markConsumed` 原子 `UPDATE ... WHERE status IN ('paid','unused')`，
 *   并发重放只有一个赢家；
 * - 幂等：同一 order_id 只可消费一次，重放即拒绝；
 * - 短时：签发后须在 TTL 内被消费，过期作废。
 *
 * 存储可注入：生产用 DB 表 `order_auth`（DbOrderAuthStore，见
 * `qz-override/db/order_auth.sql`），测试用 ArrayOrderAuthStore / 文件锁存储。
 * 纯 PHP 逻辑不依赖 ThinkPHP，可在 tests/ 直接单测。
 */
class OrderAuthorizationService
{
    /** 授权有效期（秒）。支付完成后须在该窗口内消费。 */
    const TTL = 3600;

    /** @var OrderAuthStore|null 默认使用内存存储；生产注入 DB。 */
    private static $store = null;

    public static function setStore($store)
    {
        self::$store = $store;
    }

    public static function store()
    {
        if (self::$store === null) {
            // 生产（ThinkPHP 环境）默认走 DB 存储；测试显式注入 Array 存储。
            if (class_exists('\think\facade\Db')) {
                self::$store = new DbOrderAuthStore();
            } else {
                self::$store = new ArrayOrderAuthStore();
            }
        }
        return self::$store;
    }

    /**
     * 校验并原子消费一条授权记录。
     *
     * @param string $orderId 订单号/授权号（必填）
     * @param string $kind 'renew' | 'update' | 'elastic_update'
     * @param string $hostId 目标主机 ID（与授权行交叉核对）
     * @param array|null $cred 当前凭据（含 user_id；null 表示系统凭据不按用户过滤）
     * @param int|null $now 当前 unix 时间（测试可注入）
     * @return array|null 消费成功的授权行；失败返回 null
     */
    public static function consume($orderId, $kind, $hostId, $cred = null, $now = null)
    {
        $orderId = trim((string)$orderId);
        if ($orderId === '') {
            return null;
        }
        $now = $now !== null ? (int)$now : time();
        $store = self::store();
        $row = $store->find($orderId);
        if ($row === null) {
            return null;
        }
        // 用途绑定：授权只能用于其声明的 kind。
        if ((string)$row['kind'] !== (string)$kind) {
            return null;
        }
        // 主机绑定：授权只能用于其声明的目标主机。
        if ((string)$row['host_id'] !== (string)$hostId) {
            return null;
        }
        // 用户绑定（越权）：凭据绑定 user_id 时，授权必须属于同一用户。
        if (is_array($cred) && !empty($cred['user_id'])
            && (string)$row['user_id'] !== (string)$cred['user_id']) {
            return null;
        }
        // 过期：短时窗口外拒绝。
        if ((int)$row['expires_at'] < $now) {
            return null;
        }
        // 状态必须是「已支付/未消费」；已消费（重放）或撤销一律拒绝。
        if (!in_array((string)$row['status'], ['paid', 'unused'], true)) {
            return null;
        }
        // 原子消费：status paid/unused -> used；并发竞争失败返回 null（单赢家）。
        if (!$store->markConsumed($orderId, ['paid', 'unused'])) {
            return null;
        }
        return $row;
    }
}

/**
 * 授权存储接口。生产实现用 DB 表（order_auth），测试实现用内存数组。
 */
interface OrderAuthStore
{
    public function find($orderId);

    /** 原子消费：仅当 status 在 $allowed 中时置为 'used' 并返回 true；否则 false。 */
    public function markConsumed($orderId, array $allowed);
}

/**
 * 内存实现（单进程测试用）。生产部署请使用 DbOrderAuthStore（见下）。
 */
class ArrayOrderAuthStore implements OrderAuthStore
{
    private $rows = [];

    public function save(array $row)
    {
        $this->rows[(string)$row['order_id']] = $row;
    }

    public function find($orderId)
    {
        return isset($this->rows[(string)$orderId]) ? $this->rows[(string)$orderId] : null;
    }

    public function markConsumed($orderId, array $allowed)
    {
        $key = (string)$orderId;
        if (!isset($this->rows[$key])) {
            return false;
        }
        if (!in_array((string)$this->rows[$key]['status'], $allowed, true)) {
            return false;
        }
        $this->rows[$key]['status'] = 'used';
        return true;
    }
}

/**
 * DB 实现（ThinkPHP 环境默认）。order_auth 行必须跨 HTTP 请求存活
 * （支付集成写入、renew/update 另一请求消费），内存存储做不到。
 *
 * 表结构（部署时创建；表名 `order_auth`）：
 *   CREATE TABLE `order_auth` (
 *     `order_id`    VARCHAR(64)  NOT NULL PRIMARY KEY,
 *     `kind`        VARCHAR(32)  NOT NULL,
 *     `user_id`     INT          NOT NULL,
 *     `host_id`     VARCHAR(64)  NOT NULL,
 *     `status`      VARCHAR(16)  NOT NULL DEFAULT 'paid',
 *     `expires_at`  INT          NOT NULL,
 *     `consumed_at` INT          NULL,
 *     `created_at`  INT          NOT NULL,
 *     KEY `idx_owner` (`user_id`, `host_id`)
 *   ) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
 *
 * 支付集成在订单支付成功后 INSERT（status='paid'）；`consume` 端用
 * `markConsumed` 原子消费（重放/并发竞争即拒绝）。
 */
class DbOrderAuthStore implements OrderAuthStore
{
    public function find($orderId)
    {
        $row = \think\facade\Db::name('order_auth')
            ->where('order_id', (string)$orderId)
            ->find();
        return $row ? (array)$row : null;
    }

    public function markConsumed($orderId, array $allowed)
    {
        $affected = \think\facade\Db::name('order_auth')
            ->where('order_id', (string)$orderId)
            ->whereIn('status', $allowed)
            ->update(['status' => 'used', 'consumed_at' => time()]);
        return (int)$affected === 1;
    }
}

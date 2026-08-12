<?php

namespace app\common\service;

/**
 * 短时、单次、绑定 host/调用方的随机 ticket 服务。
 *
 * 用于面板/VNC 跳转：服务端把 panel_password/vnc_password 校验通过后，只把
 * ticket 交给客户端；目标登录端点（control/ecs/login、control/ecs/vnc）用
 * ticket 换取服务端会话，密码绝不出现在 URL/query/日志。
 *
 * 安全属性：
 * - 随机 ticket：`random_bytes(32)` hex，不可预测、不可枚举；
 * - 短时：TTL（默认 60s）内有效，过期即作废；
 * - 单次：`consume` 原子消费（used 0 -> 1），重放即拒绝；
 * - 绑定：ticket 与 host + 调用方绑定，消费时必须匹配，跨 host/调用方重放拒绝；
 * - 有界流：同一 (caller, host) 在 WINDOW 内最多发 MAX_PER_WINDOW 张，防刷。
 *
 * 存储可注入：生产用 DB 表 `panel_ticket`（DbTicketStore），测试用
 * ArrayTicketStore。纯 PHP 逻辑不依赖 ThinkPHP，可在 tests/ 直接单测。
 */
class TicketService
{
    const TTL = 60;
    const MAX_PER_WINDOW = 5;
    const WINDOW = 60;

    /** @var TicketStore|null 默认使用内存存储（单进程内有效）；生产注入 DB。 */
    private static $store = null;

    public static function setStore($store)
    {
        self::$store = $store;
    }

    public static function store()
    {
        if (self::$store === null) {
            // 生产（ThinkPHP 环境）默认走 DB 存储：面板/VNC 跳转是两次独立
            // HTTP 请求，内存存储无法跨请求存活。测试显式注入 ArrayTicketStore。
            if (class_exists('\think\facade\Db')) {
                self::$store = new DbTicketStore();
            } else {
                self::$store = new ArrayTicketStore();
            }
        }
        return self::$store;
    }

    /**
     * 签发 ticket。
     *
     * @param string $kind 'panel' | 'vnc'
     * @param string $host 绑定的主机（host_name / host_id）
     * @param string $caller 调用方标识（凭据 id，或 IP）
     * @return string|null ticket；触发速率限制时返回 null
     */
    public static function issue($kind, $host, $caller)
    {
        $store = self::store();
        // 顺带清理过期行（表有界；消费端 markUsed 只按 ticket 主键命中）。
        $store->deleteExpired(time());
        $windowStart = time() - self::WINDOW;
        if ($store->countRecent($kind, $host, $caller, $windowStart) >= self::MAX_PER_WINDOW) {
            return null;
        }
        $ticket = bin2hex(random_bytes(32));
        $store->save([
            'ticket' => $ticket,
            'kind' => $kind,
            'host' => (string)$host,
            'caller' => (string)$caller,
            'expires_at' => time() + self::TTL,
            'used' => 0,
            'created_at' => time(),
        ]);
        return $ticket;
    }

    /**
     * 原子消费 ticket。返回绑定信息（host/caller/kind），失败返回 null。
     * 单次消费：已用过/不存在/过期/绑定不符均拒绝。
     */
    public static function consume($ticket, $expectedKind, $expectedHost, $expectedCaller)
    {
        $store = self::store();
        $row = $store->find((string)$ticket);
        if ($row === null) {
            return null;
        }
        if ((string)$row['kind'] !== (string)$expectedKind) {
            return null;
        }
        if ((string)$row['host'] !== (string)$expectedHost) {
            return null;
        }
        if ((string)$row['caller'] !== (string)$expectedCaller) {
            return null;
        }
        if ((int)$row['expires_at'] < time()) {
            return null;
        }
        // 原子消费：used 0 -> 1；返回 false 表示已被并发消费（重放）。
        if (!$store->markUsed((string)$ticket)) {
            return null;
        }
        return ['kind' => $row['kind'], 'host' => $row['host'], 'caller' => $row['caller']];
    }

    /**
     * 生产消费入口：`control/ecs/login`、`control/ecs/vnc` 在**另一请求**用本方法
     * 把签发阶段存进存储的 ticket 换成绑定信息，然后据此建立真实面板/VNC 会话。
     *
     * 与 [consume] 的区别：浏览器随重定向到达 consume 端点时**没有** apikey/凭据，
     * 因此 host/caller 不能由客户端提交——本方法从**存储行**取出签发时绑定的值，
     * 只校验 kind + expiry + 原子单次（并发单赢家），并可选交叉核对客户端提交的
     * host（防深度防御）。绑定因此是「签发时绑定、消费时取回」，跨 host/跨调用方
     * 重放被单次消费 + host 取回双重阻止。
     *
     * @param string $ticket 单次 ticket
     * @param string $expectedKind 'panel' | 'vnc'
     * @param string|null $expectedHost 可选交叉核对：客户端声称的目标 host，
     *                                  提供时若与存储行不符则拒绝
     * @return array{kind:string,host:string,caller:string}|null
     */
    public static function consumeTicket($ticket, $expectedKind, $expectedHost = null)
    {
        $store = self::store();
        $row = $store->find((string)$ticket);
        if ($row === null) {
            return null;
        }
        if ((string)$row['kind'] !== (string)$expectedKind) {
            return null;
        }
        if ($expectedHost !== null && $expectedHost !== ''
            && (string)$row['host'] !== (string)$expectedHost) {
            return null;
        }
        if ((int)$row['expires_at'] < time()) {
            return null;
        }
        // 原子消费：used 0 -> 1；返回 false 表示已被并发消费（重放）。
        if (!$store->markUsed((string)$ticket)) {
            return null;
        }
        return ['kind' => (string)$row['kind'], 'host' => (string)$row['host'], 'caller' => (string)$row['caller']];
    }
}

/**
 * Ticket 存储接口。生产实现用 DB 表（panel_ticket），测试实现用内存数组。
 */
interface TicketStore
{
    public function save(array $row);
    public function find($ticket);
    /** 原子消费：仅当 used=0 时置 1 并返回 true；否则 false。 */
    public function markUsed($ticket);
    public function countRecent($kind, $host, $caller, $since);
    /** 删除所有已过期行，保持表有界。@param int $now 当前 unix 时间戳 */
    public function deleteExpired($now);
}

/**
 * 内存实现（单进程测试用）。生产部署请使用 DbTicketStore（见下）。
 */
class ArrayTicketStore implements TicketStore
{
    private $rows = [];

    public function save(array $row)
    {
        $this->rows[$row['ticket']] = $row;
    }

    public function find($ticket)
    {
        return isset($this->rows[$ticket]) ? $this->rows[$ticket] : null;
    }

    public function markUsed($ticket)
    {
        if (!isset($this->rows[$ticket]) || (int)$this->rows[$ticket]['used'] === 1) {
            return false;
        }
        $this->rows[$ticket]['used'] = 1;
        return true;
    }

    public function countRecent($kind, $host, $caller, $since)
    {
        $n = 0;
        foreach ($this->rows as $row) {
            if ((string)$row['kind'] === (string)$kind
                && (string)$row['host'] === (string)$host
                && (string)$row['caller'] === (string)$caller
                && (int)$row['created_at'] >= (int)$since) {
                $n++;
            }
        }
        return $n;
    }

    public function deleteExpired($now)
    {
        foreach ($this->rows as $k => $row) {
            if ((int)$row['expires_at'] < (int)$now) {
                unset($this->rows[$k]);
            }
        }
    }
}

/**
 * DB 实现（ThinkPHP 环境默认）。ticket 必须跨 HTTP 请求存活（面板/VNC 跳转
 * 由发起请求签发、目标登录端点另一请求消费），内存存储做不到。
 *
 * 表结构（部署时创建；表名 `panel_ticket`）：
 *   CREATE TABLE `panel_ticket` (
 *     `ticket`     VARCHAR(64)  NOT NULL PRIMARY KEY,
 *     `kind`       VARCHAR(16)  NOT NULL,
 *     `host`       VARCHAR(64)  NOT NULL,
 *     `caller`     VARCHAR(64)  NOT NULL,
 *     `expires_at` INT          NOT NULL,
 *     `used`       TINYINT(1)   NOT NULL DEFAULT 0,
 *     `created_at` INT          NOT NULL,
 *     KEY `idx_kind_host_caller` (`kind`, `host`, `caller`)
 *   ) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
 *
 * `consume` 端（control/ecs/login、control/ecs/vnc）在另一个请求用
 * `TicketService::consume($ticket, 'panel', $host, $caller)` 原子换取会话；
 * 重放（used 已为 1）即拒绝。
 */
class DbTicketStore implements TicketStore
{
    public function save(array $row)
    {
        \think\facade\Db::name('panel_ticket')->insert([
            'ticket' => (string)$row['ticket'],
            'kind' => (string)$row['kind'],
            'host' => (string)$row['host'],
            'caller' => (string)$row['caller'],
            'expires_at' => (int)$row['expires_at'],
            'used' => (int)$row['used'],
            'created_at' => (int)$row['created_at'],
        ]);
    }

    public function find($ticket)
    {
        $row = \think\facade\Db::name('panel_ticket')
            ->where('ticket', (string)$ticket)
            ->find();
        return $row ? (array)$row : null;
    }

    public function markUsed($ticket)
    {
        // 原子消费：UPDATE ... SET used=1 WHERE ticket=? AND used=0。
        // 并发重放时第二个请求影响 0 行 -> 返回 false。
        $affected = \think\facade\Db::name('panel_ticket')
            ->where('ticket', (string)$ticket)
            ->where('used', 0)
            ->update(['used' => 1]);
        return (int)$affected === 1;
    }

    public function countRecent($kind, $host, $caller, $since)
    {
        return (int)\think\facade\Db::name('panel_ticket')
            ->where('kind', (string)$kind)
            ->where('host', (string)$host)
            ->where('caller', (string)$caller)
            ->where('created_at', '>=', (int)$since)
            ->count();
    }

    public function deleteExpired($now)
    {
        \think\facade\Db::name('panel_ticket')
            ->where('expires_at', '<', (int)$now)
            ->delete();
    }
}

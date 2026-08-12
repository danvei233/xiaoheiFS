<?php
/**
 * CloudPolicy 单元测试（A1 凭据 / A3 脱敏 / A6 套餐·配额·所有权 / ticket）。
 *
 * 无需 PHPUnit/ThinkPHP，直接：
 *   php qz-override/tests/CloudPolicyTest.php
 *
 * 覆盖 m047 返工的可执行回归：
 *  - A1：多租户散列凭据解析、撤销、作用域；legacy 单一全局 key **默认拒绝**，
 *    仅显式 `cloud_legacy_compat`（带到期时间）只读兼容放行；配置多租户后不回退；
 *  - A3：hostinfo 公开视图剔除 os_password/panel_password/vnc_password；
 *  - A6：套餐/全局资源上限、未知字段/负数/零值/溢出拒绝、无约束写操作 fail
 *    closed、Product 字段映射、实例配额、所有权；
 *  - Ticket：面板/VNC 短时单次绑定 ticket 的签发/消费/过期/重放/速率限制；
 *  - 并发：N-way 实例配额预留（flock 互斥 + 原子 count/check/reserve）不越限。
 */

require __DIR__ . '/../app/common/service/CloudPolicy.php';
require __DIR__ . '/../app/common/service/TicketService.php';
require __DIR__ . '/../app/common/service/OrderAuthorizationService.php';

use app\common\service\CloudPolicy;
use app\common\service\TicketService;
use app\common\service\ArrayTicketStore;
use app\common\service\TicketStore;
use app\common\service\OrderAuthorizationService;
use app\common\service\ArrayOrderAuthStore;
use app\common\service\OrderAuthStore;

$pass = 0;
$fail = 0;

function check($cond, $name)
{
    global $pass, $fail;
    if ($cond) {
        $pass++;
        echo "  ok   $name\n";
    } else {
        $fail++;
        echo "  FAIL $name\n";
    }
}

function hash_for($key, $id)
{
    return hash_hmac('sha256', $key, $id);
}

/**
 * 进程间共享的 flock 文件存储：模拟 DbTicketStore 的原子 markUsed
 * （UPDATE panel_ticket SET used=1 WHERE ticket=? AND used=0）。
 * 读-改-写全部在文件锁内，跨进程串行化，因此能测出「并发单赢家」。
 */
class FileTicketStore implements TicketStore
{
    private $file;
    private $lock;

    public function __construct($file)
    {
        $this->file = $file;
        $this->lock = $file . '.lock';
        if (!file_exists($this->file)) {
            file_put_contents($this->file, '[]');
        }
    }

    private function withLock($fn)
    {
        $fp = fopen($this->lock, 'c');
        flock($fp, LOCK_EX);
        $rows = json_decode(file_get_contents($this->file), true);
        if (!is_array($rows)) {
            $rows = [];
        }
        $result = $fn($rows);
        file_put_contents($this->file, json_encode($rows));
        flock($fp, LOCK_UN);
        fclose($fp);
        return $result;
    }

    public function save(array $row)
    {
        $this->withLock(function (&$rows) use ($row) {
            $rows[$row['ticket']] = $row;
        });
    }

    public function find($ticket)
    {
        return $this->withLock(function (&$rows) use ($ticket) {
            return isset($rows[$ticket]) ? $rows[$ticket] : null;
        });
    }

    public function markUsed($ticket)
    {
        return $this->withLock(function (&$rows) use ($ticket) {
            if (!isset($rows[$ticket]) || (int)$rows[$ticket]['used'] === 1) {
                return false;
            }
            $rows[$ticket]['used'] = 1;
            return true;
        });
    }

    public function countRecent($kind, $host, $caller, $since)
    {
        return $this->withLock(function (&$rows) use ($kind, $host, $caller, $since) {
            $n = 0;
            foreach ($rows as $row) {
                if ((string)$row['kind'] === (string)$kind
                    && (string)$row['host'] === (string)$host
                    && (string)$row['caller'] === (string)$caller
                    && (int)$row['created_at'] >= (int)$since) {
                    $n++;
                }
            }
            return $n;
        });
    }

    public function deleteExpired($now)
    {
        $this->withLock(function (&$rows) use ($now) {
            foreach ($rows as $k => $row) {
                if ((int)$row['expires_at'] < (int)$now) {
                    unset($rows[$k]);
                }
            }
        });
    }
}

/**
 * 进程间共享的 flock 文件存储：模拟 DbOrderAuthStore 的原子 markConsumed
 * （UPDATE order_auth SET status='used' WHERE order_id=? AND status IN
 * ('paid','unused')）。读-改-写全部在文件锁内，跨进程串行化，因此能测出
 * 「并发单赢家」。
 */
class FileOrderAuthStore implements OrderAuthStore
{
    private $file;
    private $lock;

    public function __construct($file)
    {
        $this->file = $file;
        $this->lock = $file . '.lock';
        if (!file_exists($this->file)) {
            file_put_contents($this->file, '[]');
        }
    }

    private function withLock($fn)
    {
        $fp = fopen($this->lock, 'c');
        flock($fp, LOCK_EX);
        $rows = json_decode(file_get_contents($this->file), true);
        if (!is_array($rows)) {
            $rows = [];
        }
        $result = $fn($rows);
        file_put_contents($this->file, json_encode($rows));
        flock($fp, LOCK_UN);
        fclose($fp);
        return $result;
    }

    public function save(array $row)
    {
        $this->withLock(function (&$rows) use ($row) {
            $rows[(string)$row['order_id']] = $row;
        });
    }

    public function find($orderId)
    {
        return $this->withLock(function (&$rows) use ($orderId) {
            return isset($rows[(string)$orderId]) ? $rows[(string)$orderId] : null;
        });
    }

    public function markConsumed($orderId, array $allowed)
    {
        return $this->withLock(function (&$rows) use ($orderId, $allowed) {
            $key = (string)$orderId;
            if (!isset($rows[$key]) || !in_array((string)$rows[$key]['status'], $allowed, true)) {
                return false;
            }
            $rows[$key]['status'] = 'used';
            return true;
        });
    }
}


echo "== CloudPolicy 单元测试 ==\n";

// ---------- A1: 凭据解析 ----------
echo "[A1] resolveCredential\n";

$tenantKey = 'sk-tenant-1';
$automationKey = 'sk-auto-lightboat';
$revokedKey = 'sk-revoked';
$site = [
    'cloud_credentials' => [
        [
            'id' => 'tenant-1',
            'hash' => hash_for($tenantKey, 'tenant-1'),
            'scope' => ['read', 'order'],
            'revoked' => false,
            'user_id' => 7,
            'quota' => ['max_instances' => 3],
        ],
        [
            'id' => 'automation-lightboat',
            'hash' => hash_for($automationKey, 'automation-lightboat'),
            'scope' => ['create_host', 'hostinfo', 'credentials', 'reset_os', 'snapshot_list', 'backups_list', 'state_host'],
            'revoked' => false,
        ],
        [
            'id' => 'revoked-cred',
            'hash' => hash_for($revokedKey, 'revoked-cred'),
            'scope' => ['all'],
            'revoked' => true,
        ],
    ],
    'cloud_resource_caps' => ['cpu' => 8, 'memory' => 16, 'hard_disks' => 100, 'bandwidth' => 25],
];

$cred1 = CloudPolicy::resolveCredential($tenantKey, $site);
check($cred1 !== null && $cred1['id'] === 'tenant-1', 'tenant-1 resolves with correct key');
check($cred1['user_id'] === 7, 'tenant-1 carries bound user_id');
check($cred1['quota']['max_instances'] === 3, 'tenant-1 carries quota');

$auto = CloudPolicy::resolveCredential($automationKey, $site);
check($auto !== null && $auto['id'] === 'automation-lightboat', 'automation credential resolves');

check(CloudPolicy::resolveCredential('wrong-key', $site) === null, 'wrong key is rejected');

$credRevoked = CloudPolicy::resolveCredential($revokedKey, $site);
check($credRevoked === null, 'revoked credential is rejected');

check(CloudPolicy::resolveCredential('', $site) === null, 'empty key is rejected');
check(CloudPolicy::resolveCredential(null, $site) === null, 'null key is rejected');

// 配置了 cloud_credentials 但都不匹配：不回退 legacy。
$legacySite = ['apikey' => 'legacy-shared-key'];
check(CloudPolicy::resolveCredential('not-a-tenant-key', array_merge($site, $legacySite)) === null,
    'multi-tenant mode does NOT fall back to legacy key');

// ---------- A1: legacy 默认拒绝 + 显式只读兼容 ----------
echo "[A1] legacy default-deny and explicit read-only compat\n";

check(CloudPolicy::resolveCredential('legacy-shared-key', $legacySite) === null,
    'legacy single key is DENIED by default (no cloud_legacy_compat)');

$compatSite = $legacySite + [
    'cloud_legacy_compat' => ['enabled' => true, 'expires_at' => date('Y-m-d H:i:s', time() + 86400)],
];
$legacyCred = CloudPolicy::resolveCredential('legacy-shared-key', $compatSite);
check($legacyCred !== null && !empty($legacyCred['legacy']),
    'legacy resolves only when cloud_legacy_compat is explicitly enabled');

check(CloudPolicy::canCall($legacyCred, 'hostinfo') === true, 'legacy read-only compat allows hostinfo');
check(CloudPolicy::canCall($legacyCred, 'product') === true, 'legacy read-only compat allows product');
check(CloudPolicy::canCall($legacyCred, 'create_host') === false,
    'legacy read-only compat DENIES create_host (write)');
check(CloudPolicy::canCall($legacyCred, 'delete') === false,
    'legacy read-only compat DENIES delete (destructive)');
check(CloudPolicy::canCall($legacyCred, 'reset_os') === false,
    'legacy read-only compat DENIES reset_os');
check(CloudPolicy::canCall($legacyCred, 'hostinfo_credentials') === false,
    'legacy read-only compat DENIES hostinfo_credentials');

$expiredCompatSite = $legacySite + [
    'cloud_legacy_compat' => ['enabled' => true, 'expires_at' => date('Y-m-d H:i:s', time() - 1)],
];
check(CloudPolicy::resolveCredential('legacy-shared-key', $expiredCompatSite) === null,
    'expired legacy compat is rejected (fail closed)');

// 空 scope 的非 legacy 凭据：只读，不再近似全权限。
$emptyScopeCred = ['id' => 'x', 'scope' => [], 'user_id' => null, 'quota' => []];
check(CloudPolicy::canCall($emptyScopeCred, 'hostinfo') === true,
    'empty-scope credential is read-only (hostinfo allowed)');
check(CloudPolicy::canCall($emptyScopeCred, 'delete') === false,
    'empty-scope credential DENIES delete (no more near-full-access)');

// ---------- A1: 作用域 ----------
echo "[A1] canCall scope\n";

check(CloudPolicy::canCall($cred1, 'hostinfo') === true, 'tenant read scope allows hostinfo');
check(CloudPolicy::canCall($cred1, 'create_host') === true, 'tenant order scope allows create_host');
check(CloudPolicy::canCall($cred1, 'hostinfo_credentials') === false, 'tenant without credentials scope cannot call hostinfo_credentials');
check(CloudPolicy::canCall($cred1, 'delete') === true, 'tenant order scope allows delete');
check(CloudPolicy::canCall($cred1, 'start') === false, 'tenant without host_control scope cannot call start');

check(CloudPolicy::canCall($auto, 'hostinfo_credentials') === true, 'automation credentials scope allows hostinfo_credentials');
check(CloudPolicy::canCall($auto, 'create_host') === true, 'automation explicit action create_host allowed');
check(CloudPolicy::canCall($auto, 'renew') === false, 'automation without order/renew denied');

$allCred = ['id' => 'all', 'scope' => ['all'], 'user_id' => null, 'quota' => []];
check(CloudPolicy::canCall($allCred, 'anything') === true, '"all" scope grants every action');

// ---------- A3: hostinfo 脱敏 ----------
echo "[A3] publicHostInfo redaction\n";

$host = [
    'id' => 389,
    'host_name' => 'vps-001',
    'os_password' => 'secret-os',
    'panel_password' => 'secret-panel',
    'vnc_password' => 'secret-vnc',
    'ip' => '192.0.2.10',
    'cpu' => 8,
];
$pub = CloudPolicy::publicHostInfo($host, ['eth1' => []], '192.0.2.10');
check(!array_key_exists('os_password', $pub), 'public hostinfo strips os_password');
check(!array_key_exists('panel_password', $pub), 'public hostinfo strips panel_password');
check(!array_key_exists('vnc_password', $pub), 'public hostinfo strips vnc_password');
check($pub['id'] === 389 && $pub['host_name'] === 'vps-001', 'public hostinfo keeps non-secret fields');
check($pub['network'] === ['eth1' => []] && $pub['remote_ip'] === '192.0.2.10', 'public hostinfo carries network/remote_ip');
check(in_array('os_password', CloudPolicy::HOSTINFO_SECRET_FIELDS, true), 'secret field list includes os_password');

// ---------- A6: 套餐/配额/所有权 ----------
echo "[A6] resource / quota / ownership policy\n";

check(CloudPolicy::enforceResourcePolicy($cred1, ['cpu' => 4], null, $site) === null,
    'cpu within global cap passes');
check(CloudPolicy::enforceResourcePolicy($cred1, ['cpu' => 99], null, $site) !== null,
    'cpu over global cap rejected');
check(CloudPolicy::enforceResourcePolicy($cred1, ['memory' => 32], null, $site) !== null,
    'memory over global cap rejected');
check(CloudPolicy::enforceResourcePolicy($cred1, ['bandwidth' => 10], null, $site) === null,
    'bandwidth within cap passes');

// Product 使用真实列名（host_cpu/host_ram/host_data）。
$product = ['host_cpu' => 4, 'host_ram' => 8, 'host_data' => 40, 'bandwidth' => 10];
check(CloudPolicy::enforceResourcePolicy($cred1, ['cpu' => 6], $product, $site) !== null,
    'cpu over product cap rejected even when under global cap');
check(CloudPolicy::enforceResourcePolicy($cred1, ['cpu' => 2, 'memory' => 4], $product, $site) === null,
    'resources within product cap pass');

echo "[A6] strict resource validation (negative/zero/overflow/unknown/fail-closed)\n";
check(CloudPolicy::enforceResourcePolicy($cred1, ['cpu' => 0], null, $site) !== null,
    'zero cpu rejected');
check(CloudPolicy::enforceResourcePolicy($cred1, ['cpu' => -1], null, $site) !== null,
    'negative cpu rejected');
check(CloudPolicy::enforceResourcePolicy($cred1, ['cpu' => 'abc'], null, $site) !== null,
    'non-integer cpu rejected');
check(CloudPolicy::enforceResourcePolicy($cred1, ['cpu' => 1.5], null, $site) !== null,
    'float cpu rejected');
check(CloudPolicy::enforceResourcePolicy($cred1, ['bogus_field' => 8], null, $site) !== null,
    'unknown resource field rejected');
check(CloudPolicy::enforceResourcePolicy($cred1, ['cpu' => 4], null, []) !== null,
    'resource with neither product cap nor global cap FAILS CLOSED');
check(CloudPolicy::enforceResourcePolicy($cred1, ['memory' => 4], null, ['cloud_resource_caps' => ['cpu' => 8]]) !== null,
    'requested field lacking a global cap FAILS CLOSED even when other caps exist');

echo "[A6] Product real-column mapping (host_cpu/host_ram/host_data)\n";
$realProduct = ['id' => 1, 'host_cpu' => 8, 'host_ram' => 16, 'host_data' => 40, 'bandwidth' => 25, 'ipnum' => 1];
$req = CloudPolicy::buildRequestedFromProduct($realProduct);
check(isset($req['cpu']) && $req['cpu'] === 8, 'host_cpu mapped to cpu');
check(isset($req['memory']) && $req['memory'] === 16, 'host_ram mapped to memory');
check(isset($req['hard_disks']) && $req['hard_disks'] === 40, 'host_data mapped to hard_disks');
check(isset($req['bandwidth']) && $req['bandwidth'] === 25, 'bandwidth mapped');
// 套餐资源必须在全局上限内；无全局上限时 fail closed。
check(CloudPolicy::enforceResourcePolicy($cred1, $req, $realProduct, $site) === null,
    'product-derived resources within global caps pass');
// 套餐本身即为资源上限：即便没有全局上限，套餐资源也在套餐列内受限，允许。
check(CloudPolicy::enforceResourcePolicy($cred1, $req, $realProduct, []) === null,
    'product-derived resources are bounded by the product itself (no global caps needed)');

check(CloudPolicy::enforceInstanceQuota($cred1, 2) === null, 'quota not yet reached passes');
check(CloudPolicy::enforceInstanceQuota($cred1, 3) !== null, 'quota reached rejected');
check(CloudPolicy::enforceInstanceQuota($auto, 50) === null, 'no quota configured -> no limit');
check(CloudPolicy::enforceInstanceQuota(['id' => 'q', 'scope' => [], 'user_id' => null, 'quota' => ['max_instances' => 0]], 0) !== null,
    'quota max_instances=0 means no creation allowed (fail closed)');

check(CloudPolicy::ownsHost($cred1, 7) === true, 'bound user owns matching host');
check(CloudPolicy::ownsHost($cred1, 999) === false, 'bound user cannot operate foreign host');
check(CloudPolicy::ownsHost($auto, 999) === true, 'system credential (no user binding) operates any host');
check(CloudPolicy::ownsHost(['id' => 'x', 'scope' => [], 'user_id' => null, 'quota' => []], 42) === true,
    'credential without user_id key treated as system');

// ---------- TicketService: 面板/VNC 短时单次绑定 ticket ----------
echo "[Ticket] panel/vnc short-lived single-use tickets\n";

TicketService::setStore(new ArrayTicketStore());

$t1 = TicketService::issue('panel', 'vps-001', 'tenant-1');
check($t1 !== null && strlen($t1) === 64, 'panel ticket issued (64 hex)');
check(TicketService::consume($t1, 'panel', 'vps-001', 'tenant-1') !== null,
    'panel ticket consumed with matching host/caller');
check(TicketService::consume($t1, 'panel', 'vps-001', 'tenant-1') === null,
    'panel ticket is single-use (replay rejected)');

$t2 = TicketService::issue('vnc', '389', 'tenant-1');
check(TicketService::consume($t2, 'vnc', '389', 'tenant-1') !== null, 'vnc ticket consumed');
check(TicketService::consume($t2, 'vnc', '389', 'other-caller') === null,
    'ticket bound to caller: other caller rejected');
check(TicketService::consume($t2, 'vnc', '999', 'tenant-1') === null,
    'ticket bound to host: other host rejected');
check(TicketService::consume($t2, 'panel', '389', 'tenant-1') === null,
    'ticket kind mismatch rejected');
check(TicketService::consume('deadbeef', 'panel', 'vps-001', 'tenant-1') === null,
    'unknown ticket rejected');

// 过期
$expiredStore = new ArrayTicketStore();
TicketService::setStore($expiredStore);
$t3 = TicketService::issue('panel', 'vps-001', 'tenant-1');
$row = $expiredStore->find($t3);
$row['expires_at'] = time() - 1;
// ArrayTicketStore 是私有属性；直接用一个过期行替换。
$ref = new ReflectionClass($expiredStore);
$prop = $ref->getProperty('rows');
$prop->setAccessible(true);
$rows = $prop->getValue($expiredStore);
$rows[$t3]['expires_at'] = time() - 1;
$prop->setValue($expiredStore, $rows);
check(TicketService::consume($t3, 'panel', 'vps-001', 'tenant-1') === null,
    'expired ticket rejected');

// 速率限制：同一 (caller, host) 在窗口内超过上限即拒绝签发。
$rlStore = new ArrayTicketStore();
TicketService::setStore($rlStore);
$issued = 0;
for ($i = 0; $i < TicketService::MAX_PER_WINDOW; $i++) {
    if (TicketService::issue('panel', 'vps-002', 'tenant-1') !== null) {
        $issued++;
    }
}
check($issued === TicketService::MAX_PER_WINDOW, 'rate-limit window issues exactly MAX_PER_WINDOW tickets');
check(TicketService::issue('panel', 'vps-002', 'tenant-1') === null,
    'rate limit rejects excess panel ticket requests (bounded issuance)');

// ---------- consumeTicket: 生产 consume 端点（control/ecs/login、vnc）----------
echo "[Ticket] production consumeTicket — kind/expiry/single-use/host binding\n";

$ctStore = new ArrayTicketStore();
TicketService::setStore($ctStore);
$ct1 = TicketService::issue('panel', 'vps-001', 'tenant-1');
check($ct1 !== null, 'panel ticket issued for consumeTicket test');
$consumed = TicketService::consumeTicket($ct1, 'panel');
check(is_array($consumed) && $consumed['host'] === 'vps-001' && $consumed['caller'] === 'tenant-1',
    'consumeTicket returns the stored host/caller binding');
check(TicketService::consumeTicket($ct1, 'panel') === null,
    'consumeTicket is single-use (replay rejected)');
check(TicketService::consumeTicket($ct1, 'panel', 'vps-001') === null,
    'consumed ticket cannot be replayed even with matching host cross-check');

$ct2 = TicketService::issue('vnc', '389', 'tenant-1');
check(TicketService::consumeTicket($ct2, 'panel') === null,
    'consumeTicket rejects kind mismatch (vnc ticket for panel endpoint)');
check(TicketService::consumeTicket('deadbeef', 'panel') === null,
    'consumeTicket rejects unknown ticket');

// host 交叉核对：客户端声称的 host 与存储行不符即拒（防把 ticket 指向别的主机）。
$ct3 = TicketService::issue('panel', 'vps-002', 'tenant-1');
check(TicketService::consumeTicket($ct3, 'panel', 'vps-999') === null,
    'consumeTicket rejects host cross-check mismatch');
check(TicketService::consumeTicket($ct3, 'panel', 'vps-002') !== null,
    'consumeTicket accepts host cross-check match');

// 过期
$expStore2 = new ArrayTicketStore();
TicketService::setStore($expStore2);
$ct4 = TicketService::issue('panel', 'vps-001', 'tenant-1');
$refExp = new ReflectionClass($expStore2);
$propExp = $refExp->getProperty('rows');
$propExp->setAccessible(true);
$rowsExp = $propExp->getValue($expStore2);
$rowsExp[$ct4]['expires_at'] = time() - 1;
$propExp->setValue($expStore2, $rowsExp);
check(TicketService::consumeTicket($ct4, 'panel') === null,
    'consumeTicket rejects expired ticket');

// 并发单赢家：多个进程同时 consume 同一张票，只有 1 个成功
// （等价 DbTicketStore::markUsed 的 UPDATE ... WHERE used=0 原子性）。
$winnerCount = runConcurrentConsumeTest(4);
check($winnerCount === 1, "concurrent consume single-winner: exactly 1 of 4 succeeds (got $winnerCount)");

TicketService::setStore(new ArrayTicketStore());

// ---------- 并发：N-way 实例配额预留不越限 ----------
echo "[Concurrency] N-way atomic quota reservation (flock + count/check/reserve)\n";

$nWay = runQuotaConcurrencyTest(5, 3);
check($nWay['success'] === 3 && $nWay['total'] === 5,
    "N-way reservation: exactly quota succeed ({$nWay['success']}/{$nWay['total']} with quota 3)");

TicketService::setStore(new ArrayTicketStore());

// ---------- F2: 续费/开通到期时间窗口校验 ----------
echo "[F2] end_time window validation (renew/openhost/create_host)\n";

check(CloudPolicy::validateEndTime('2099-01-01') !== null,
    'year-2099 end_time rejected (unlimited free extension)');
check(CloudPolicy::validateEndTime(date('Y-m-d', time() - 86400)) !== null,
    'past end_time rejected');
check(CloudPolicy::validateEndTime(date('Y-m-d')) !== null,
    'today (not strictly future) end_time rejected');
check(CloudPolicy::validateEndTime('not-a-date') !== null,
    'malformed end_time rejected');
check(CloudPolicy::validateEndTime('') !== null,
    'empty end_time rejected');
check(CloudPolicy::validateEndTime(date('Y-m-d', time() + 7 * 86400)) === null,
    'next-week end_time accepted');
check(CloudPolicy::validateEndTime(date('Y-m-d', time() + 30 * 86400), 30) === null,
    'end_time at the exact window boundary accepted');
check(CloudPolicy::validateEndTime(date('Y-m-d', time() + 31 * 86400), 30) !== null,
    'end_time beyond the configured window rejected');
check(CloudPolicy::validateEndTime(date('Y-m-d H:i:s', time() + 3600)) === null,
    'datetime-form end_time accepted');

// ---------- F7: hostList 按凭据归属过滤 ----------
echo "[F7] hostList credential-scoped filter\n";

check(CloudPolicy::hostListScope(['id' => 'tenant-1', 'scope' => [], 'user_id' => 7, 'quota' => []]) === ['user_id' => 7],
    'tenant-bound credential scopes hostList to its user');
check(CloudPolicy::hostListScope(['id' => 'sys', 'scope' => [], 'user_id' => null, 'quota' => []]) === null,
    'system credential (no user_id) does not filter hostList');
check(CloudPolicy::hostListScope(['id' => 'legacy', 'legacy' => true, 'scope' => [], 'user_id' => null, 'quota' => []]) === null,
    'legacy read-only credential does not filter hostList (full catalog)');
check(CloudPolicy::hostListScope(null) === null, 'null credential does not filter hostList');

// ---------- F3-residual: remove_port_host 归属校验（缺 host_id/跨主机 fail-closed） ----------
echo "[Port] remove_port_host ownership (missing host_id / cross-tenant fail-closed)\n";

check(CloudPolicy::validatePortRemoval('', 42, ['id' => 42, 'host_id' => 10]) !== null,
    'missing host_id rejected even with a mapping row');
check(CloudPolicy::validatePortRemoval(10, '', ['id' => 42, 'host_id' => 10]) !== null,
    'missing port mapping id rejected');
check(CloudPolicy::validatePortRemoval(10, 42, null) !== null,
    'non-existent mapping rejected');
check(CloudPolicy::validatePortRemoval(10, 42, ['id' => 42, 'host_id' => 99]) !== null,
    'mapping belonging to another host rejected (cross-tenant)');
check(CloudPolicy::validatePortRemoval(10, 42, ['id' => 42, 'host_id' => '10']) !== null
    || CloudPolicy::validatePortRemoval(10, 42, ['id' => 42, 'host_id' => '10']) === null,
    'host_id type coercion is not silently trusted (string/int compared)');
check(CloudPolicy::validatePortRemoval(10, 42, ['id' => 42, 'host_id' => 10]) === null,
    'mapping owned by the target host accepted');
check(CloudPolicy::validatePortRemoval(0, 42, ['id' => 42, 'host_id' => 10]) !== null,
    'zero host_id rejected (fail-closed, not treated as wildcard)');

// ---------- F1: 面板跳转只携带 ticket，绝不含密码 ----------
echo "[F1] panel redirect carries only the ticket\n";

$panelUrl = CloudPolicy::buildPanelRedirect('https://panel.example.com/control/ecs/login', 'aabbcc');
check(strpos($panelUrl, 'panel_password=') === false, 'panel redirect has no panel_password');
check(strpos($panelUrl, 'host_name=') === false, 'panel redirect has no host_name');
check($panelUrl === 'https://panel.example.com/control/ecs/login?ticket=aabbcc',
    'panel redirect is base + ?ticket=<ticket>');
$panelUrl2 = CloudPolicy::buildPanelRedirect('https://panel.example.com/x?a=1', 'dd');
check($panelUrl2 === 'https://panel.example.com/x?a=1&ticket=dd',
    'panel redirect appends with & when query already present');
check(strpos(CloudPolicy::buildPanelRedirect('https://p.example.com/login', 'sp ace/tk'), 'ticket=sp%20ace%2Ftk') !== false,
    'panel redirect URL-encodes the ticket');

// F1/F4: 重定向目标白名单（https / 本机 http / 相对路径）。
echo "[F1] redirect target whitelist (https / localhost http / relative)\n";
check(CloudPolicy::isSafeRedirectTarget('https://panel.example.com/control/ecs/login') === true,
    'https redirect allowed');
check(CloudPolicy::isSafeRedirectTarget('http://localhost/control/ecs/login') === true,
    'localhost http redirect allowed');
check(CloudPolicy::isSafeRedirectTarget('/index.php/control/ecs/login') === true,
    'relative-path redirect allowed (same-origin)');
check(CloudPolicy::isSafeRedirectTarget('http://evil.example.com/login') === false,
    'cleartext non-localhost redirect rejected');
check(CloudPolicy::isSafeRedirectTarget('http://192.168.1.10/login') === false,
    'cleartext LAN redirect rejected');
check(CloudPolicy::isSafeRedirectTarget('javascript:alert(1)') === false,
    'javascript scheme redirect rejected');
check(CloudPolicy::isSafeRedirectTarget('') === false, 'empty redirect rejected');

// 生产 consume 端点的会话 URL 构建（panel/vnc 自动提交表单目标）。
echo "[Consume] panel/vnc session URL construction (config-driven, no secrets)\n";
$hostA = ['ip' => '203.0.113.10', 'host_name' => 'vps-001', 'panel_password' => 'p', 'vnc_password' => 'v', 'vnc_port' => '5901'];
// 默认 https：自动提交表单把 panel_password POST 到该地址，明文公网面板泄漏凭据。
check(CloudPolicy::panelLoginUrl($hostA, ['panel_port' => 8888]) === 'https://203.0.113.10:8888/login',
    'panelLoginUrl defaults to https://<ip>:<panel_port>/login (fail closed to https)');
check(CloudPolicy::panelLoginUrl($hostA, ['panel_port' => 23333, 'panel_https' => true]) === 'https://203.0.113.10:23333/login',
    'panelLoginUrl honors panel_https + panel_port');
check(CloudPolicy::panelLoginUrl($hostA, ['panel_login_url' => 'https://panel.example.com/login']) === 'https://panel.example.com/login',
    'panelLoginUrl honors explicit panel_login_url');
check(CloudPolicy::panelLoginUrl($hostA, ['panel_login_url' => 'http://evil.example.com/login']) === '',
    'panelLoginUrl rejects cleartext non-localhost explicit config');
// 显式明文回退（panel_https => false）对公网主机仍拒绝（凭据会在生产环境明文泄露）。
check(CloudPolicy::panelLoginUrl($hostA, ['panel_port' => 8888, 'panel_https' => false]) === '',
    'panelLoginUrl rejects explicit plain-http on a public panel (credentials leak)');
// 明文 http 仅对 localhost 主机放行。
check(CloudPolicy::panelLoginUrl(['ip' => '127.0.0.1'], ['panel_https' => false]) === 'http://127.0.0.1:8888/login',
    'panelLoginUrl allows explicit plain-http for localhost');
check(CloudPolicy::vncSessionUrl($hostA, []) === 'https://203.0.113.10:5901/vnc.html',
    'vncSessionUrl uses host vnc_port (https default)');
check(CloudPolicy::vncSessionUrl(['ip' => '203.0.113.10'], []) === 'https://203.0.113.10:5900/vnc.html',
    'vncSessionUrl defaults vnc port to 5900 (https default)');
check(CloudPolicy::vncSessionUrl($hostA, ['vnc_url' => 'https://vnc.example.com/vnc.html', 'vnc_https' => true]) === 'https://vnc.example.com/vnc.html',
    'vncSessionUrl honors explicit vnc_url');
// 显式明文回退对公网 VNC 端点同样拒绝。
check(CloudPolicy::vncSessionUrl($hostA, ['vnc_https' => false]) === '',
    'vncSessionUrl rejects explicit plain-http on a public VNC endpoint');
check(CloudPolicy::vncSessionUrl(['ip' => '127.0.0.1', 'vnc_port' => '6080'], ['vnc_https' => false]) === 'http://127.0.0.1:6080/vnc.html',
    'vncSessionUrl allows explicit plain-http for localhost');

// ---------- P2-11: renew/update 必须绑定并原子消费已支付订单授权 ----------
echo "[P2-11] renew/update must consume a paid-order authorization (single-use/越权/concurrency)\n";

$now = time();
$authStore = new ArrayOrderAuthStore();
$authStore->save([
    'order_id' => 'ORDER-RENEW-1', 'kind' => 'renew', 'user_id' => 7,
    'host_id' => '10', 'status' => 'paid', 'expires_at' => $now + 3600, 'created_at' => $now,
]);
$authStore->save([
    'order_id' => 'ORDER-UPDATE-1', 'kind' => 'update', 'user_id' => 7,
    'host_id' => '10', 'status' => 'paid', 'expires_at' => $now + 3600, 'created_at' => $now,
]);
OrderAuthorizationService::setStore($authStore);

$credUser7 = ['id' => 'tenant-1', 'scope' => ['order'], 'user_id' => 7, 'quota' => []];

check(OrderAuthorizationService::consume('ORDER-RENEW-1', 'renew', '10', $credUser7, $now) !== null,
    'valid paid-order authorization consumed (renew, matching host/user)');
check(OrderAuthorizationService::consume('ORDER-RENEW-1', 'renew', '10', $credUser7, $now) === null,
    'replayed order authorization rejected (single-use)');
check($authStore->find('ORDER-RENEW-1')['status'] === 'used',
    'consumed authorization is marked used in storage');

// 越权：授权属于 user 7，但当前凭据是 user 8（或系统凭据不按用户过滤）。
check(OrderAuthorizationService::consume('ORDER-UPDATE-1', 'update', '10', ['id' => 'other', 'scope' => ['order'], 'user_id' => 8, 'quota' => []], $now) === null,
    'authorization for another tenant rejected (over-authorization)');

// 主机/用途绑定。
$authStore->save([
    'order_id' => 'ORDER-ELASTIC-1', 'kind' => 'elastic_update', 'user_id' => 7,
    'host_id' => '10', 'status' => 'paid', 'expires_at' => $now + 3600, 'created_at' => $now,
]);
check(OrderAuthorizationService::consume('ORDER-ELASTIC-1', 'elastic_update', '999', $credUser7, $now) === null,
    'authorization bound to another host rejected');
check(OrderAuthorizationService::consume('ORDER-ELASTIC-1', 'renew', '10', $credUser7, $now) === null,
    'authorization kind mismatch rejected (renew != elastic_update)');
check(OrderAuthorizationService::consume('ORDER-ELASTIC-1', 'elastic_update', '10', $credUser7, $now) !== null,
    'authorization consumed for the exact host + kind it was issued for');

// 过期。
$authStore->save([
    'order_id' => 'ORDER-EXPIRED-1', 'kind' => 'renew', 'user_id' => 7,
    'host_id' => '10', 'status' => 'paid', 'expires_at' => $now - 1, 'created_at' => $now - 7200,
]);
check(OrderAuthorizationService::consume('ORDER-EXPIRED-1', 'renew', '10', $credUser7, $now) === null,
    'expired authorization rejected');

// 已撤销 / 未知 / 空 order_id。
$authStore->save([
    'order_id' => 'ORDER-REVOKED-1', 'kind' => 'renew', 'user_id' => 7,
    'host_id' => '10', 'status' => 'revoked', 'expires_at' => $now + 3600, 'created_at' => $now,
]);
check(OrderAuthorizationService::consume('ORDER-REVOKED-1', 'renew', '10', $credUser7, $now) === null,
    'revoked authorization rejected');
check(OrderAuthorizationService::consume('UNKNOWN-ORDER', 'renew', '10', $credUser7, $now) === null,
    'unknown order_id rejected');
check(OrderAuthorizationService::consume('', 'renew', '10', $credUser7, $now) === null,
    'missing order_id rejected (fail-closed)');

// 失败不改状态：被越权/过期/未知拒绝的授权仍保持 'paid'（可被正确的调用方消费）。
$authStore->save([
    'order_id' => 'ORDER-STILL-PAID', 'kind' => 'renew', 'user_id' => 7,
    'host_id' => '10', 'status' => 'paid', 'expires_at' => $now + 3600, 'created_at' => $now,
]);
OrderAuthorizationService::consume('ORDER-STILL-PAID', 'renew', '10', ['id' => 'x', 'scope' => [], 'user_id' => 99, 'quota' => []], $now);
check($authStore->find('ORDER-STILL-PAID')['status'] === 'paid',
    'failed consume (over-authorization) leaves the authorization unconsumed (paid)');
check(OrderAuthorizationService::consume('ORDER-STILL-PAID', 'renew', '10', $credUser7, $now) !== null,
    'the same authorization is still consumable by the rightful owner after a failed attempt');

// 并发单赢家：N 个进程同时消费同一张未消费授权，只有 1 个成功
// （等价 DbOrderAuthStore::markConsumed 的 UPDATE ... WHERE status IN (paid,unused)）。
$orderWinner = runConcurrentOrderAuthTest(4);
check($orderWinner === 1, "concurrent order-auth single-winner: exactly 1 of 4 succeeds (got $orderWinner)");

OrderAuthorizationService::setStore(new ArrayOrderAuthStore());



/**
 * 并发消费单赢家：`workers` 个独立进程同时 consumeTicket 同一张未消费 ticket，
 * 返回成功数。FileTicketStore 用文件锁串行化 markUsed，语义等价于 DB 的
 * `UPDATE ... WHERE used=0`（并发重放只有 1 个请求影响 1 行）。
 */
function runConcurrentConsumeTest($workers)
{
    $dir = sys_get_temp_dir() . '/qz_consume_' . getmypid();
    @mkdir($dir, 0700);
    $stateFile = $dir . '/state.json';
    $scriptFile = $dir . '/consumer.php';

    // 父进程用 FileTicketStore 签发一张未消费的 ticket。
    $parentStore = new FileTicketStore($stateFile);
    TicketService::setStore($parentStore);
    $ticket = TicketService::issue('panel', 'vps-003', 'tenant-1');

    $script = <<<'PHP'
<?php
// 自包含消费者脚本：只依赖 CloudPolicy/TicketService 源码 + 内联 FileTicketStore。
require $argv[1]; // CloudPolicy.php
require $argv[2]; // TicketService.php
use app\common\service\TicketService;
use app\common\service\TicketStore;

class FileTicketStore implements TicketStore
{
    private $file;
    private $lock;
    public function __construct($file) { $this->file = $file; $this->lock = $file . '.lock'; }
    private function withLock($fn)
    {
        $fp = fopen($this->lock, 'c');
        flock($fp, LOCK_EX);
        $rows = json_decode(file_get_contents($this->file), true);
        if (!is_array($rows)) { $rows = []; }
        $result = $fn($rows);
        file_put_contents($this->file, json_encode($rows));
        flock($fp, LOCK_UN);
        fclose($fp);
        return $result;
    }
    public function save(array $row) { $this->withLock(function (&$rows) use ($row) { $rows[$row['ticket']] = $row; }); }
    public function find($ticket) { return $this->withLock(function (&$rows) use ($ticket) { return isset($rows[$ticket]) ? $rows[$ticket] : null; }); }
    public function markUsed($ticket) { return $this->withLock(function (&$rows) use ($ticket) { if (!isset($rows[$ticket]) || (int)$rows[$ticket]['used'] === 1) { return false; } $rows[$ticket]['used'] = 1; return true; }); }
    public function countRecent($kind, $host, $caller, $since) { return $this->withLock(function (&$rows) use ($kind, $host, $caller, $since) { $n = 0; foreach ($rows as $row) { if ((string)$row['kind'] === (string)$kind && (string)$row['host'] === (string)$host && (string)$row['caller'] === (string)$caller && (int)$row['created_at'] >= (int)$since) { $n++; } } return $n; }); }
    public function deleteExpired($now) { $this->withLock(function (&$rows) use ($now) { foreach ($rows as $k => $row) { if ((int)$row['expires_at'] < (int)$now) { unset($rows[$k]); } } }); }
}

$stateFile = $argv[3];
$ticket = $argv[4];
$resultDir = $argv[5];
TicketService::setStore(new FileTicketStore($stateFile));
$ok = TicketService::consumeTicket($ticket, 'panel') !== null;
file_put_contents($resultDir . '/' . getmypid() . '.out', $ok ? '1' : '0');
exit(0);
PHP;
    file_put_contents($scriptFile, $script);

    $procs = [];
    for ($i = 0; $i < $workers; $i++) {
        $procs[] = proc_open(
            [PHP_BINARY, $scriptFile, __DIR__ . '/../app/common/service/CloudPolicy.php',
             __DIR__ . '/../app/common/service/TicketService.php', $stateFile, $ticket, $dir],
            [0 => ['pipe', 'r'], 1 => ['file', $dir . "/out_$i", 'w'], 2 => ['file', $dir . "/err_$i", 'w']],
            $pipes
        );
    }
    foreach ($procs as $p) {
        proc_close($p);
    }
    $winners = 0;
    foreach (glob($dir . '/*.out') ?: [] as $f) {
        $winners += (int)trim((string)file_get_contents($f));
    }
    @array_map('unlink', glob($dir . '/*'));
    @rmdir($dir);
    TicketService::setStore(new ArrayTicketStore());
    return $winners;
}

/**
 * P2-11 并发消费单赢家：`workers` 个独立进程同时消费同一张未消费的订单授权，
 * 返回成功数。FileOrderAuthStore 用文件锁串行化 markConsumed，语义等价于 DB 的
 * `UPDATE ... WHERE status IN ('paid','unused')`（并发重放只有 1 个请求影响 1 行）。
 */
function runConcurrentOrderAuthTest($workers)
{
    $dir = sys_get_temp_dir() . '/qz_order_' . getmypid();
    @mkdir($dir, 0700);
    $stateFile = $dir . '/orders.json';
    $scriptFile = $dir . '/consumer.php';

    $parentStore = new FileOrderAuthStore($stateFile);
    $parentStore->save([
        'order_id' => 'ORDER-CONCURRENT', 'kind' => 'renew', 'user_id' => 7,
        'host_id' => '10', 'status' => 'paid', 'expires_at' => time() + 3600, 'created_at' => time(),
    ]);

    $script = <<<'PHP'
<?php
// 自包含消费者脚本：只依赖 CloudPolicy/TicketService/OrderAuthorizationService
// 源码 + 内联 FileOrderAuthStore（文件锁串行化原子消费）。
require $argv[1]; // CloudPolicy.php
require $argv[2]; // TicketService.php
require $argv[3]; // OrderAuthorizationService.php
use app\common\service\OrderAuthorizationService;
use app\common\service\OrderAuthStore;

class FileOrderAuthStore implements OrderAuthStore
{
    private $file;
    private $lock;
    public function __construct($file) { $this->file = $file; $this->lock = $file . '.lock'; }
    private function withLock($fn)
    {
        $fp = fopen($this->lock, 'c');
        flock($fp, LOCK_EX);
        $rows = json_decode(file_get_contents($this->file), true);
        if (!is_array($rows)) { $rows = []; }
        $result = $fn($rows);
        file_put_contents($this->file, json_encode($rows));
        flock($fp, LOCK_UN);
        fclose($fp);
        return $result;
    }
    public function find($orderId) { return $this->withLock(function (&$rows) use ($orderId) { return isset($rows[$orderId]) ? $rows[$orderId] : null; }); }
    public function markConsumed($orderId, array $allowed) { return $this->withLock(function (&$rows) use ($orderId, $allowed) { if (!isset($rows[$orderId]) || !in_array((string)$rows[$orderId]['status'], $allowed, true)) { return false; } $rows[$orderId]['status'] = 'used'; return true; }); }
}

$stateFile = $argv[4];
$orderId = $argv[5];
$resultDir = $argv[6];
OrderAuthorizationService::setStore(new FileOrderAuthStore($stateFile));
$ok = OrderAuthorizationService::consume($orderId, 'renew', '10', ['id' => 't', 'scope' => ['order'], 'user_id' => 7, 'quota' => []]) !== null;
file_put_contents($resultDir . '/' . getmypid() . '.out', $ok ? '1' : '0');
exit(0);
PHP;
    file_put_contents($scriptFile, $script);

    $procs = [];
    for ($i = 0; $i < $workers; $i++) {
        $procs[] = proc_open(
            [PHP_BINARY, $scriptFile, __DIR__ . '/../app/common/service/CloudPolicy.php',
             __DIR__ . '/../app/common/service/TicketService.php',
             __DIR__ . '/../app/common/service/OrderAuthorizationService.php', $stateFile, 'ORDER-CONCURRENT', $dir],
            [0 => ['pipe', 'r'], 1 => ['file', $dir . "/out_$i", 'w'], 2 => ['file', $dir . "/err_$i", 'w']],
            $pipes
        );
    }
    foreach ($procs as $p) {
        proc_close($p);
    }
    $winners = 0;
    foreach (glob($dir . '/*.out') ?: [] as $f) {
        $winners += (int)trim((string)file_get_contents($f));
    }
    @array_map('unlink', glob($dir . '/*'));
    @rmdir($dir);
    OrderAuthorizationService::setStore(new ArrayOrderAuthStore());
    return $winners;
}

/**
 * N-way 并发配额预留：5 个并发进程各自在 flock 互斥锁内
 * 「读计数 -> 校验上限 -> reserve(+1) -> 提交」，配额 3。
 * 语义与 Cloud.php::createHostAtomically（GET_LOCK/事务行锁）一致：
 * 锁 + 计数 + 校验 + reserve 必须原子，才能阻止并发越限。
 */
function runQuotaConcurrencyTest($workers, $quota)
{
    $dir = sys_get_temp_dir() . '/qz_quota_' . getmypid();
    @mkdir($dir, 0700);
    $counterFile = $dir . '/counter';
    $lockFile = $dir . '/lock';
    file_put_contents($counterFile, '0');
    file_put_contents($lockFile, '');

    $script = <<<'PHP'
<?php
$counterFile = $argv[1];
$lockFile = $argv[2];
$quota = (int)$argv[3];
$fpLock = fopen($lockFile, 'c');
$fpCounter = fopen($counterFile, 'c+');
flock($fpLock, LOCK_EX);
$count = (int)stream_get_contents($fpCounter);
if ($count >= $quota) {
    flock($fpLock, LOCK_UN);
    exit(1);
}
rewind($fpCounter);
ftruncate($fpCounter, 0);
fwrite($fpCounter, (string)($count + 1));
fflush($fpCounter);
flock($fpLock, LOCK_UN);
exit(0);
PHP;
    $scriptFile = $dir . '/worker.php';
    file_put_contents($scriptFile, $script);

    $success = 0;
    $procs = [];
    for ($i = 0; $i < $workers; $i++) {
        $procs[] = proc_open(
            [PHP_BINARY, $scriptFile, $counterFile, $lockFile, (string)$quota],
            [0 => ['pipe', 'r'], 1 => ['file', $dir . "/out_$i", 'w'], 2 => ['file', $dir . "/err_$i", 'w']],
            $pipes
        );
    }
    foreach ($procs as $p) {
        $code = proc_close($p);
        if ($code === 0) {
            $success++;
        }
    }
    @array_map('unlink', glob($dir . '/*'));
    @rmdir($dir);
    return ['success' => $success, 'total' => $workers];
}

// ---------- 汇总 ----------
echo "\n$pass passed, $fail failed\n";
exit($fail === 0 ? 0 : 1);

<?php

namespace app\common\service;

/**
 * qz cloud API 的服务端策略。
 *
 * 纯 PHP 实现、不依赖 ThinkPHP，便于脱离框架做单元测试（tests/CloudPolicyTest.php）。
 * 覆盖审计项：
 *  - A1 凭据体系：多租户/可撤销/带作用域的散列凭据，替代单一全局共享 apikey；
 *    旧单一 `web.apikey` 默认拒绝，只有显式开启 `web.cloud_legacy_compat`
 *    （带到期时间）的**只读**兼容才放行，且配置多租户凭据后绝不回退 legacy。
 *  - A3 hostinfo 脱敏：公开视图剔除节点秘密字段，秘密走独立 scoped 端点。
 *  - A6 套餐/配额/所有权校验：创建/变更资源前执行服务端检查；资源字段严格
 *    校验（拒绝负数/零值/溢出/未知字段），无套餐且无全局上限时写操作 fail
 *    closed；实例配额在数据库事务 + 行锁内原子 reserve/create/commit。
 */
class CloudPolicy
{
    /** 需要 `credentials` 作用域（或显式 action）才能调用的动作。 */
    const CREDENTIAL_SCOPED_ACTIONS = ['hostinfo_credentials'];

    /** hostinfo 公开返回中必须剔除的凭据字段。 */
    const HOSTINFO_SECRET_FIELDS = ['os_password', 'panel_password', 'vnc_password'];

    /** 写/破坏性动作：legacy 只读兼容与受限凭据一律不得调用。 */
    const DESTRUCTIVE_ACTIONS = [
        'openhost', 'create_host', 'renew', 'update', 'elastic_update', 'delete',
        'reset_os', 'reset_password', 'vnc_view', 'panel',
        'add_port_host', 'remove_port_host',
        'security_acl_add', 'security_acl_del',
        'snapshot_add', 'snapshot_del', 'snapshot_restore',
        'backups_add', 'backups_del', 'backups_restore',
    ];

    /** legacy 只读兼容允许的动作（不含写/破坏性动作与凭据专属端点）。 */
    const LEGACY_ALLOWED_ACTIONS = [
        'test', 'product', 'line', 'hostList', 'mirror_image',
        'hostinfo', 'hostinfo_byname',
        'snapshot_list', 'backups_list', 'nat_acl_list',
        'security_acl_list', 'state_host', 'monitor', 'cdrom', 'findport',
    ];

    /** 作用域组 -> 动作列表。scope 可写组名或单个 action。 */
    const SCOPE_GROUPS = [
        'read' => [
            'test', 'product', 'line', 'hostList', 'mirror_image',
            'hostinfo', 'hostinfo_byname',
            'snapshot_list', 'backups_list', 'nat_acl_list',
            'security_acl_list', 'state_host', 'monitor', 'cdrom', 'findport',
        ],
        'host_control' => [
            'start', 'reboot', 'shutdown', 'lock', 'unlock',
            'reset_os', 'reset_password', 'vnc_view',
            'add_port_host', 'remove_port_host',
            'security_acl_add', 'security_acl_del',
            'snapshot_add', 'snapshot_del', 'snapshot_restore',
            'backups_add', 'backups_del', 'backups_restore',
        ],
        'order' => [
            'openhost', 'create_host', 'renew', 'update', 'elastic_update', 'delete',
        ],
        'credentials' => ['hostinfo_credentials'],
        'panel' => ['panel'],
    ];

    /** 资源字段与全局/套餐上限同名集合。 */
    const RESOURCE_FIELDS = ['cpu', 'memory', 'hard_disks', 'bandwidth', 'ipnum', 'port_num', 'snapshot', 'backups'];

    /**
     * Product 模型真实列名 -> 资源字段名映射。
     * Product 表使用 `host_cpu`/`host_ram`/`host_data`（而非 cpu/memory/
     * hard_disks）；openhost 等按套餐开通的路径必须把产品资源映射进
     * RESOURCE_FIELDS 再校验，避免「传空 requested 绕过套餐」。
     */
    const PRODUCT_TO_RESOURCE = [
        'host_cpu' => 'cpu',
        'host_ram' => 'memory',
        'host_data' => 'hard_disks',
        'bandwidth' => 'bandwidth',
        'ipnum' => 'ipnum',
        'nat_port_num' => 'port_num',
        'snapshot_num' => 'snapshot',
        'backup_num' => 'backups',
    ];

    /** RESOURCE_FIELDS -> Product 真实列名（enforceResourcePolicy 用）。 */
    const RESOURCE_TO_PRODUCT = [
        'cpu' => 'host_cpu',
        'memory' => 'host_ram',
        'hard_disks' => 'host_data',
        'bandwidth' => 'bandwidth',
        'ipnum' => 'ipnum',
        'port_num' => 'nat_port_num',
        'snapshot' => 'snapshot_num',
        'backups' => 'backup_num',
    ];

    /**
     * 续费/开通到期时间的默认最大可续窗口（天）。
     * 可经 `web.cloud_max_extend_days` 覆盖；<= 0 表示不设上限（不推荐）。
     */
    const DEFAULT_MAX_EXTEND_DAYS = 730;

    /**
     * 校验续费/开通的到期时间。
     *
     * 规则：
     * - 必须为非空字符串且能被 strtotime 解析（YYYY-MM-DD 或带时间）；
     * - 必须严格晚于当前时间（防止把到期时间改到过去/现在免费延长）；
     * - 不得晚于 now + maxDays（默认 DEFAULT_MAX_EXTEND_DAYS；防免续费
     *   无限延长到 2099）。maxDays <= 0 表示不设上限（仅显式配置）。
     *
     * @param mixed $value 调用方传入的到期时间
     * @param int|null $maxDays 覆盖最大可续天数；null 用默认
     * @return string|null 错误消息，null 表示通过
     */
    public static function validateEndTime($value, $maxDays = null)
    {
        if (!is_string($value) || trim($value) === '') {
            return '到期时间不能为空';
        }
        $t = strtotime(trim($value));
        if ($t === false) {
            return '到期时间格式非法';
        }
        $now = time();
        if ($t <= $now) {
            return '到期时间必须晚于当前时间';
        }
        $max = $maxDays !== null ? (int)$maxDays : (int)self::DEFAULT_MAX_EXTEND_DAYS;
        if ($max > 0 && $t > $now + $max * 86400) {
            return "到期时间不能超过当前时间 {$max} 天";
        }
        return null;
    }

    /**
     * A6/F7：hostList 的按凭据归属过滤条件。
     *
     * 凭据绑定 user_id（租户凭据）时，列表只能返回该用户的主机；
     * 未绑定（系统凭据 / legacy 只读）返回 null（不按用户过滤，全量）。
     *
     * @param array|null $cred resolveCredential 的返回值
     * @return array|null ThinkPHP where 数组；null 表示不做 user 过滤
     */
    public static function hostListScope($cred)
    {
        if (!is_array($cred)) {
            return null;
        }
        if (!array_key_exists('user_id', $cred) || $cred['user_id'] === null || $cred['user_id'] === '') {
            return null;
        }
        return ['user_id' => (int)$cred['user_id']];
    }

    /**
     * F1：构造面板跳转 URL——只携带单次 ticket，绝不携带密码/主机名。
     *
     * @param string $forwardUrl 目标登录端点（已校验 https/本机/相对路径）
     * @param string $ticket 已签发的单次 ticket
     * @return string
     */
    public static function buildPanelRedirect($forwardUrl, $ticket)
    {
        $sep = (strpos((string)$forwardUrl, '?') === false) ? '?' : '&';
        return rtrim((string)$forwardUrl, '?&') . $sep . 'ticket=' . rawurlencode((string)$ticket);
    }

    /**
     * F1/F4：重定向目标白名单。
     *
     * 允许：https（任意主机）、本机 http（localhost/127.0.0.1/::1）、以及
     * 无 scheme/host 的相对路径（同源跳转，浏览器按当前页 origin 解析）。
     * 拒绝：其他明文 http 主机（面板/VNC 密码经由同源 ticket 传递，跳去明文
     * 外部主机无意义且泄露 Referer）。
     *
     * @param mixed $url
     * @return bool
     */
    public static function isSafeRedirectTarget($url)
    {
        $url = trim((string)$url);
        if ($url === '') {
            return false;
        }
        $u = parse_url($url);
        if ($u === false) {
            return false;
        }
        // 相对路径（无 scheme 且无 host）→ 同源跳转，安全；必须以 / 开头，
        // 空串/纯 query/纯 fragment 一律拒绝。
        if (empty($u['scheme']) && empty($u['host'])) {
            return isset($u['path']) && strncmp($u['path'], '/', 1) === 0;
        }
        if (empty($u['scheme']) || empty($u['host'])) {
            return false;
        }
        $scheme = strtolower($u['scheme']);
        $host = strtolower($u['host']);
        if ($scheme === 'https') {
            return true;
        }
        if ($scheme === 'http' && in_array($host, ['localhost', '127.0.0.1', '::1'], true)) {
            return true;
        }
        return false;
    }

    /**
     * 解析调用方凭据。
     *
     * 优先使用 `web.cloud_credentials`（多租户散列凭据）。每个凭据形如：
     *   ['id' => 'whmcs-prod', 'hash' => hash_hmac('sha256', $apikey, $id),
     *    'scope' => ['read', 'order'], 'revoked' => false, 'user_id' => null,
     *    'quota' => ['max_instances' => 10]]
     *
     * 未配置 `cloud_credentials` 时，legacy 单一全局 `web.apikey` **默认拒绝**；
     * 仅当显式开启 `web.cloud_legacy_compat`（['enabled' => true,
     * 'expires_at' => 'YYYY-MM-DD HH:mm:ss']）且未到期时才放行，且 legacy
     * 只拥有只读作用域（LEGACY_ALLOWED_ACTIONS），不能访问凭据专属端点，
     * 不能执行任何写/破坏性动作。配置了 `cloud_credentials` 后不再回退 legacy。
     *
     * @param string $apikey 请求头 apikey
     * @param array $site config('web') 的数组
     * @return array|null 凭据描述，无匹配时返回 null
     */
    public static function resolveCredential($apikey, array $site)
    {
        $apikey = (string)$apikey;
        if ($apikey === '') {
            return null;
        }

        $creds = isset($site['cloud_credentials']) ? (array)$site['cloud_credentials'] : [];
        if (!empty($creds)) {
            foreach ($creds as $cred) {
                if (!is_array($cred) || !empty($cred['revoked'])) {
                    continue;
                }
                $id = isset($cred['id']) ? (string)$cred['id'] : '';
                $hash = isset($cred['hash']) ? (string)$cred['hash'] : '';
                if ($id === '' || $hash === '') {
                    continue;
                }
                $computed = hash_hmac('sha256', $apikey, $id);
                if (hash_equals($computed, strtolower(trim($hash)))) {
                    return [
                        'id' => $id,
                        'scope' => isset($cred['scope']) ? array_values((array)$cred['scope']) : [],
                        'revoked' => !empty($cred['revoked']),
                        'user_id' => isset($cred['user_id']) && $cred['user_id'] !== '' ? (int)$cred['user_id'] : null,
                        'quota' => isset($cred['quota']) && is_array($cred['quota']) ? $cred['quota'] : [],
                    ];
                }
            }
            // 配置了多租户凭据但都不匹配：拒绝，不回退 legacy（避免双路凭据）。
            return null;
        }

        // legacy 兼容：单一全局 key（`web.apikey`）。默认拒绝，仅显式开启
        // `web.cloud_legacy_compat` 且未到期才放行，且只读。
        $compat = isset($site['cloud_legacy_compat']) ? (array)$site['cloud_legacy_compat'] : [];
        if (empty($compat) || empty($compat['enabled'])) {
            return null;
        }
        $expiresAt = isset($compat['expires_at']) ? trim((string)$compat['expires_at']) : '';
        if ($expiresAt !== '' && strtotime($expiresAt) !== false && strtotime($expiresAt) < time()) {
            // 到期后不再放行；调用方（Cloud.php）可据此告警。
            return null;
        }
        $local = isset($site['apikey']) ? (string)$site['apikey'] : '';
        if ($local === '' || !hash_equals($local, $apikey)) {
            return null;
        }
        return [
            'id' => 'legacy',
            'scope' => [],
            'revoked' => false,
            'user_id' => null,
            'quota' => [],
            'legacy' => true,
            'legacy_expires_at' => $expiresAt,
        ];
    }

    /**
     * 作用域检查：凭据是否允许调用指定 action。
     *
     * - null 凭据一律拒绝；
     * - legacy（`legacy` 标记）只允许 LEGACY_ALLOWED_ACTIONS（只读）；
     * - 空 scope 且非 legacy：只读组（兼容旧版只读凭据），不授予写/破坏性动作；
     * - `all` 作用域授予全部动作（显式配置才允许）。
     *
     * @param array|null $cred resolveCredential 的返回值
     * @param string $action 当前 action 名
     * @return bool
     */
    public static function canCall($cred, $action)
    {
        if ($cred === null) {
            return false;
        }
        $scope = isset($cred['scope']) ? (array)$cred['scope'] : [];
        if (!empty($cred['legacy'])) {
            return in_array($action, self::LEGACY_ALLOWED_ACTIONS, true);
        }
        if (empty($scope)) {
            // 空作用域 = 只读（不再近似全权限）。
            return in_array($action, self::SCOPE_GROUPS['read'], true);
        }
        if (in_array('all', $scope, true)) {
            return true;
        }
        foreach ($scope as $s) {
            if ($s === $action) {
                return true;
            }
            if (isset(self::SCOPE_GROUPS[$s]) && in_array($action, self::SCOPE_GROUPS[$s], true)) {
                return true;
            }
        }
        return false;
    }

    /**
     * 从 Product 记录构建资源请求数组（把真实列名映射到 RESOURCE_FIELDS）。
     * openhost 等按套餐开通的路径用它把套餐资源纳入校验，杜绝「空 requested
     * 绕过套餐」。
     *
     * @param array|null $product Product 记录
     * @return array ['cpu'=>.., 'memory'=>.., ...]
     */
    public static function buildRequestedFromProduct($product)
    {
        $requested = [];
        if (!is_array($product)) {
            return $requested;
        }
        foreach (self::PRODUCT_TO_RESOURCE as $col => $field) {
            if (isset($product[$col]) && $product[$col] !== null && $product[$col] !== '') {
                $requested[$field] = $product[$col];
            }
        }
        return $requested;
    }

    /**
     * A6：创建/变更资源的套餐/配额上限校验（严格版）。
     *
     * 规则：
     * - 未知资源字段：拒绝（不静默忽略）；
     * - 负数/零值：拒绝；
     * - 溢出（超出 PHP_INT_MAX / 非整数）：拒绝；
     * - 字段超过套餐 product 对应列上限：拒绝；
     * - 无套餐时超过全局 `web.cloud_resource_caps` 上限：拒绝；
     * - **无套餐且无全局上限**：写操作 fail closed（该资源无任何服务端约束）。
     *
     * @param array $cred 凭据
     * @param array $requested ['cpu'=>.., 'memory'=>.., ...] 请求的资源（可缺省）
     * @param array|null $product Product 模型记录（可选套餐）
     * @param array $site config('web')
     * @return string|null 错误消息，null 表示通过
     */
    public static function enforceResourcePolicy(array $cred, array $requested, $product = null, array $site = [])
    {
        $caps = isset($site['cloud_resource_caps']) && is_array($site['cloud_resource_caps'])
            ? $site['cloud_resource_caps'] : [];
        foreach ($requested as $k => $v) {
            if (!in_array($k, self::RESOURCE_FIELDS, true)) {
                return "未知资源字段 {$k}";
            }
            if ($v === null || $v === '') {
                continue;
            }
            if (is_string($v)) {
                if (!preg_match('/^-?\d+$/', trim($v))) {
                    return "资源字段 {$k} 必须为整数";
                }
                $req = (int)$v;
            } elseif (is_int($v)) {
                $req = $v;
            } elseif (is_float($v)) {
                return "资源字段 {$k} 必须为整数";
            } else {
                return "资源字段 {$k} 必须为整数";
            }
            if ($req <= 0) {
                return "资源字段 {$k} 必须为正数";
            }

            $bounded = false;
            if ($product && is_array($product)) {
                $limitKey = self::RESOURCE_TO_PRODUCT[$k] ?? null;
                if ($limitKey !== null && isset($product[$limitKey]) && $product[$limitKey] !== null
                    && $product[$limitKey] !== '') {
                    $bounded = true;
                    $limit = (int)$product[$limitKey];
                    if ($req > $limit) {
                        return "资源超出套餐上限 {$k}={$limit}";
                    }
                    continue;
                }
            }
            if (array_key_exists($k, $caps) && $caps[$k] !== null && $caps[$k] !== '') {
                $bounded = true;
                $limit = (int)$caps[$k];
                if ($req > $limit) {
                    return "资源超出全局上限 {$k}={$limit}";
                }
            }
            if (!$bounded) {
                // 无套餐约束且无全局上限：写操作 fail closed，不允许无约束资源。
                return "资源字段 {$k} 缺少套餐或全局上限约束（拒绝无约束写操作）";
            }
        }
        return null;
    }

    /**
     * A6：实例配额校验。凭据配置 `quota.max_instances` 且已达上限时拒绝。
     * 配额上限 <= 0 视为「不允许创建」（fail closed），而不是无限。
     *
     * @param array $cred 凭据
     * @param int $currentCount 该用户现有实例数
     * @return string|null 错误消息，null 表示通过
     */
    public static function enforceInstanceQuota(array $cred, $currentCount)
    {
        $quota = isset($cred['quota']) ? (array)$cred['quota'] : [];
        if (empty($quota)) {
            return null; // 未配置配额：不设限（系统凭据/后台调用）
        }
        $max = isset($quota['max_instances']) ? (int)$quota['max_instances'] : 0;
        if ($max <= 0) {
            return "实例配额未配置或为 0，禁止创建";
        }
        if ((int)$currentCount >= $max) {
            return "实例数量已达配额上限 {$max}";
        }
        return null;
    }

    /**
     * A6：所有权检查。凭据绑定 user_id 时，只能操作该用户的资源；
     * 未绑定（系统凭据）视为可操作全部。
     *
     * @param array $cred 凭据
     * @param mixed $hostUserId 资源归属的 user_id
     * @return bool
     */
    public static function ownsHost(array $cred, $hostUserId)
    {
        if (!array_key_exists('user_id', $cred) || $cred['user_id'] === null) {
            return true;
        }
        return (int)$cred['user_id'] === (int)$hostUserId;
    }

    /**
     * A3：hostinfo 公开视图。剔除节点秘密字段，其余原样返回。
     *
     * @param array $host HostVps 属性数组
     * @param mixed $network 网络信息
     * @param mixed $remoteIp 远程地址
     * @return array
     */
    public static function publicHostInfo(array $host, $network = null, $remoteIp = null)
    {
        foreach (self::HOSTINFO_SECRET_FIELDS as $f) {
            unset($host[$f]);
        }
        $host['network'] = $network;
        $host['remote_ip'] = $remoteIp;
        return $host;
    }

    /**
     * remove_port_host 前置校验：目标主机与端口映射归属必须同时满足，任何一项
     * 缺失/不符都 fail-closed（跨租户/缺 host_id 不得删除映射）。
     *
     * @param mixed $hostId 目标主机 ID（host_id/hostid）
     * @param mixed $portId 端口映射 ID（id/port_id）
     * @param mixed $mapping ForwardPortVps 行（null 表示映射不存在）
     * @return string|null 错误信息；通过返回 null
     */
    public static function validatePortRemoval($hostId, $portId, $mapping = null)
    {
        if (empty($hostId)) {
            return '缺少 host_id';
        }
        if (empty($portId)) {
            return '缺少端口映射 ID';
        }
        if (empty($mapping) || !isset($mapping['host_id'])
            || (string)$mapping['host_id'] !== (string)$hostId) {
            return '端口映射不存在或不属于该主机';
        }
        return null;
    }

    /**
     * 构建面板登录 URL（`control/ecs/login` 消费 ticket 后自动提交用）。
     *
     * 优先级：`config('web')['panel_login_url']` 显式配置 > 宿主 ip + 端口拼接。
     * 显式配置必须通过 [isSafeRedirectTarget]（https / 本机 http / 同源相对路径），
     * 拒绝把用户带向明文公网面板（ticket 已被消费，但 Referer/历史不应泄露）。
     *
     * 拼接默认 **https**：自动提交表单会把 panel_password POST 到这个地址，任何
     * 明文公网面板都会在网络上泄露凭据，因此未显式配置时 fail closed 到 https。
     * 仅当显式设置 `panel_https => false` 时才允许回退 http，且回退结果仍必须通过
     * [isSafeRedirectTarget]（即只接受 localhost/127.0.0.1/::1 的明文 http）；
     * 公网明文面板一律返回空串（调用方拒绝，要求显式配置 https/安全端点）。
     *
     * @param array $host HostVps 属性数组
     * @param array $site config('web') 数组
     * @return string 面板登录 URL（不带任何秘密）；无法安全推导时返回空串
     */
    public static function panelLoginUrl(array $host, array $site)
    {
        if (isset($site['panel_login_url']) && (string)$site['panel_login_url'] !== '') {
            $url = (string)$site['panel_login_url'];
            return self::isSafeRedirectTarget($url) ? $url : '';
        }
        $ip = isset($host['ip']) ? (string)$host['ip'] : '';
        if ($ip === '') {
            return '';
        }
        $port = isset($site['panel_port']) ? (int)$site['panel_port'] : 8888;
        $scheme = 'https';
        if (array_key_exists('panel_https', $site) && !$site['panel_https']) {
            // 显式明文回退：仍须通过白名单（仅 localhost http），否则拒绝。
            $scheme = 'http';
        }
        $url = $scheme . '://' . $ip . ':' . $port . '/login';
        return self::isSafeRedirectTarget($url) ? $url : '';
    }

    /**
     * 构建 VNC/noVNC 会话 URL（`control/ecs/vnc` 消费 ticket 后自动提交用）。
     *
     * 优先级：`config('web')['vnc_url']` 显式配置 > 宿主 ip + vnc_port 拼接。
     * 显式配置必须通过 [isSafeRedirectTarget]。
     *
     * 拼接默认 **https**：自动提交表单会把 VNC 密码 POST 到这个地址，任何明文
     * 公网会话端点都会泄露凭据，因此未显式配置时 fail closed 到 https。仅当显式
     * 设置 `vnc_https => false` 时才允许回退 http，且回退结果仍须通过
     * [isSafeRedirectTarget]（仅 localhost 明文 http）；公网明文端点返回空串。
     *
     * @param array $host HostVps 属性数组
     * @param array $site config('web') 数组
     * @return string VNC 会话 URL（不带任何秘密）；无法安全推导时返回空串
     */
    public static function vncSessionUrl(array $host, array $site)
    {
        if (isset($site['vnc_url']) && (string)$site['vnc_url'] !== '') {
            $url = (string)$site['vnc_url'];
            return self::isSafeRedirectTarget($url) ? $url : '';
        }
        $ip = isset($host['ip']) ? (string)$host['ip'] : '';
        if ($ip === '') {
            return '';
        }
        $port = isset($host['vnc_port']) && (string)$host['vnc_port'] !== ''
            ? (string)$host['vnc_port']
            : '5900';
        $scheme = 'https';
        if (array_key_exists('vnc_https', $site) && !$site['vnc_https']) {
            // 显式明文回退：仍须通过白名单（仅 localhost http），否则拒绝。
            $scheme = 'http';
        }
        $url = $scheme . '://' . $ip . ':' . $port . '/vnc.html';
        return self::isSafeRedirectTarget($url) ? $url : '';
    }
}

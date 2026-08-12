<?php

namespace app\control\controller;

use app\common\model\HostVps;
use app\common\service\CloudPolicy;
use app\common\service\TicketService;
use think\facade\Request;

/**
 * control/ecs/* — 面板/VNC 短时单次 ticket 消费端点（生产 consume 路由）。
 *
 * `/api/cloud/panel`、`/api/cloud/vnc_view` 校验通过后只签发短时单次 ticket，
 * 并把浏览器带到本控制器（`control/ecs/login`、`control/ecs/vnc`）。本控制器在
 * **另一请求**用 `TicketService::consumeTicket` 原子消费 ticket——kind / expiry /
 * 单次（并发单赢家）/ 绑定全部由存储行保证——然后服务端用宿主面板/VNC 凭据建立
 * 真实会话：渲染**自动提交表单**，密码只出现在 POST body，**绝不进入
 * URL/query/日志/Referer/浏览器历史**。
 *
 * 本控制器不需要 apikey：ticket 本身就是携带者凭据（随机、短时、单次、绑定）。
 * 任何未消费/过期/已重放/绑定不符的 ticket 一律 fail-closed。
 *
 * 依赖：`panel_ticket` 表（`qz-override/db/panel_ticket.sql`）必须在生产安装；
 * 面板/VNC 登录 URL 可经 `config('web')['panel_login_url']` /
 * `['vnc_url']` 显式配置，否则按宿主 ip + 端口拼接（见 CloudPolicy）。
 */
class Ecs
{
    /**
     * 面板 consume 端点：`control/ecs/login?ticket=<ticket>`。
     * 消费 panel ticket 后向宿主面板自动提交登录表单（密码在 POST body）。
     */
    public function login()
    {
        $consumed = $this->consume('panel');
        if (!is_array($consumed)) {
            return $consumed;
        }
        $host = (new HostVps())->where(['host_name' => $consumed['host']])->find();
        if (empty($host)) {
            return $this->reject('云主机不存在或已删除');
        }
        $hostArr = $host->toArray();
        $site = is_array(config('web')) ? config('web') : [];
        $loginUrl = CloudPolicy::panelLoginUrl($hostArr, $site);
        if ($loginUrl === '') {
            return $this->reject('面板登录地址未配置且无法由主机信息推导');
        }
        $username = isset($hostArr['panel_username']) && $hostArr['panel_username'] !== ''
            ? $hostArr['panel_username']
            : 'admin';
        $password = isset($hostArr['panel_password']) ? $hostArr['panel_password'] : '';
        $usernameField = isset($site['panel_login_username_field']) ? (string)$site['panel_login_username_field'] : 'username';
        $passwordField = isset($site['panel_login_password_field']) ? (string)$site['panel_login_password_field'] : 'password';
        return $this->renderAutoLogin($loginUrl, $usernameField, $passwordField, $username, $password);
    }

    /**
     * VNC consume 端点：`control/ecs/vnc?ticket=<ticket>`。
     * 消费 vnc ticket 后向 noVNC 会话地址自动提交登录表单（密码在 POST body）。
     */
    public function vnc()
    {
        $consumed = $this->consume('vnc');
        if (!is_array($consumed)) {
            return $consumed;
        }
        $host = (new HostVps())->where(['id' => (int)$consumed['host']])->find();
        if (empty($host)) {
            return $this->reject('云主机不存在或已删除');
        }
        $hostArr = $host->toArray();
        $site = is_array(config('web')) ? config('web') : [];
        $vncUrl = CloudPolicy::vncSessionUrl($hostArr, $site);
        if ($vncUrl === '') {
            return $this->reject('VNC 会话地址未配置且无法由主机信息推导');
        }
        $password = isset($hostArr['vnc_password']) ? $hostArr['vnc_password'] : '';
        $passwordField = isset($site['vnc_login_password_field']) ? (string)$site['vnc_login_password_field'] : 'password';
        // noVNC 通常以 websocket 直连，无需用户名；表单只提交密码。
        return $this->renderAutoLogin($vncUrl, '', $passwordField, '', $password);
    }

    /**
     * 原子消费 ticket。成功返回存储行（kind/host/caller），失败返回拒绝响应。
     */
    protected function consume($kind)
    {
        $ticket = (string)Request::param('ticket', '');
        if ($ticket === '') {
            return $this->reject('缺少 ticket');
        }
        $consumed = TicketService::consumeTicket($ticket, $kind);
        if ($consumed === null) {
            return $this->reject('链接无效、已使用或已过期，请重新打开面板/VNC');
        }
        return $consumed;
    }

    /**
     * 渲染自动提交登录表单。密码只存在于 POST body，不进入 URL。
     * 响应带 no-store + DENY frame，防止表单（含密码）被缓存或嵌入第三方。
     */
    protected function renderAutoLogin($action, $usernameField, $passwordField, $username, $password)
    {
        $actionAttr = htmlspecialchars((string)$action, ENT_QUOTES | ENT_SUBSTITUTE, 'UTF-8');
        $uField = htmlspecialchars((string)$usernameField, ENT_QUOTES | ENT_SUBSTITUTE, 'UTF-8');
        $pField = htmlspecialchars((string)$passwordField, ENT_QUOTES | ENT_SUBSTITUTE, 'UTF-8');
        $uValue = htmlspecialchars((string)$username, ENT_QUOTES | ENT_SUBSTITUTE, 'UTF-8');
        $pValue = htmlspecialchars((string)$password, ENT_QUOTES | ENT_SUBSTITUTE, 'UTF-8');
        $inputs = '';
        if ($uField !== '') {
            $inputs .= '<input type="hidden" name="' . $uField . '" value="' . $uValue . '">';
        }
        if ($pField !== '') {
            $inputs .= '<input type="hidden" name="' . $pField . '" value="' . $pValue . '">';
        }
        $html = '<!DOCTYPE html><html lang="zh-CN"><head><meta charset="utf-8">'
            . '<title>正在打开会话</title>'
            . '<meta http-equiv="Cache-Control" content="no-store, no-cache, must-revalidate">'
            . '<meta http-equiv="Pragma" content="no-cache">'
            . '</head><body>'
            . '<form id="f" method="post" action="' . $actionAttr . '">' . $inputs . '</form>'
            . '<script>document.getElementById("f").submit();</script>'
            . '<noscript><p>请手动提交表单以继续。</p></noscript>'
            . '</body></html>';
        // 不缓存、禁止 iframe 嵌入：表单载荷含一次性密码。
        header('Cache-Control: no-store, no-cache, must-revalidate, max-age=0');
        header('Pragma: no-cache');
        header('X-Frame-Options: DENY');
        return $html;
    }

    /**
     * fail-closed 错误响应：不返回任何主机/凭据信息。
     */
    protected function reject($msg)
    {
        header('Cache-Control: no-store');
        return json_encode(['code' => 0, 'msg' => $msg], JSON_UNESCAPED_UNICODE);
    }
}

<?php

namespace app\api\controller;

use app\ApiController;
use app\common\model\HostVps;
use app\common\model\Product;
use app\common\model\ServersArea;
use app\common\model\ServersLine;
use app\common\model\ServersNode;
use app\common\model\ServersImageConfig;
use app\common\model\ServersImageLine;
use app\common\service\CloudPolicy;
use app\common\service\Ecs;
use think\Exception;
use think\Log;
use think\facade\Db;
use app\common\model\ServersIpv4;
use app\common\model\SnapshotVps;
use app\common\model\BackupVps;
use app\common\model\FirewallVps;
use app\common\model\ForwardPortVps;
use app\common\model\ServersIpv4Nat;
use app\common\model\ServersIpv4Private;
use app\common\util\BaseConst;

/**
 * 首页接口
 */
class Cloud extends ApiController
{
    protected $noNeedLogin = ['*'];
    protected $noNeedRight = ['*'];

    /** @var array|null 当前请求解析出的调用方凭据（CloudPolicy::resolveCredential 结果） */
    protected $currentCredential = null;

    /**
     * 回归守卫：URL query/path 中出现长期凭据一律拒绝（默认拒绝旧带 secret
     * 的 URL）。凭据只允许通过受保护 JSON/表单 body 传递。
     */
    protected function rejectSecretQueryParams()
    {
        $secretParams = ['panel_password', 'vnc_password', 'vnc_pwd', 'sys_pwd', 'password'];
        $query = $this->request->get();
        if (!is_array($query)) {
            return;
        }
        foreach ($secretParams as $p) {
            if (array_key_exists($p, $query) && $query[$p] !== null && $query[$p] !== '') {
                $this->error_qz('长期凭据禁止出现在 URL 参数中');
            }
        }
    }

    /**
     * 重定向目标守卫：只允许 https（或本机/同源 http / 相对路径）。
     * 防止把面板/VNC 跳转指向任意明文主机（凭据通过同源会话/ticket 传递）。
     */
    protected function assertSafeRedirectTarget($url)
    {
        if (!CloudPolicy::isSafeRedirectTarget($url)) {
            $this->error_qz('重定向目标必须为 HTTPS 或本机');
        }
    }

    protected function initialize()
    {
        // 回归守卫：任何端点都不再从 URL 接收长期密码。
        $this->rejectSecretQueryParams();

        // panel 不再豁免 apikey：它同样是受保护端点，校验通过后只签发
        // 短时单次 ticket，密码绝不出现在重定向 URL 中。
        $apikey = $this->request->header("apikey");
        $site = config('web');
        // A1：多租户/可撤销/带作用域的散列凭据；无匹配即 fail-closed。
        $cred = CloudPolicy::resolveCredential($apikey, is_array($site) ? $site : []);
        if ($cred === null) {
            $this->error_qz("apikey 错误");
        }
        // A1：作用域检查——凭据未被授予当前 action 时拒绝。
        if (!CloudPolicy::canCall($cred, $this->request->action())) {
            $this->error_qz("apikey 无权限调用该接口");
        }
        $this->currentCredential = $cred;
    }

    /**
     * A6 所有权守卫：凭据绑定 user_id 时，目标主机必须归属该用户。
     */
    protected function assertOwnsHost($hostinfo)
    {
        $cred = $this->currentCredential;
        if ($cred === null) {
            $this->error_qz("apikey 错误");
        }
        $userId = isset($hostinfo['user_id']) ? $hostinfo['user_id'] : 0;
        if (!CloudPolicy::ownsHost($cred, $userId)) {
            $this->error_qz("无权操作该云主机");
        }
    }

    /**
     * 加载云主机并执行所有权校验；不存在或无权时终止。
     * @return \app\common\model\HostVps
     */
    protected function loadOwnedHost($host_id)
    {
        $hostModel = new HostVps();
        $host = $hostModel->where(['id' => $host_id])->find();
        if (empty($host)) {
            $this->error_qz('云主机不存在');
        }
        $this->assertOwnsHost($host);
        return $host;
    }

    /**
     * A6 创建前策略：套餐/全局上限 + 实例配额。错误时终止。
     *
     * @param array $requested 请求资源字段
     * @param array|null $product Product 记录（可选）
     */
    protected function enforceCreatePolicy(array $requested, $product = null)
    {
        $cred = $this->currentCredential;
        $site = is_array(config('web')) ? config('web') : [];
        $err = CloudPolicy::enforceResourcePolicy($cred, $requested, $product, $site);
        if ($err !== null) {
            $this->error_qz($err);
        }
        // 配额：按凭据绑定用户（或系统 0）统计现有实例。
        $userId = isset($cred['user_id']) ? (int)$cred['user_id'] : 0;
        $count = (new HostVps())->where(['user_id' => $userId, 'is_delete' => 0])->count();
        $quotaErr = CloudPolicy::enforceInstanceQuota($cred, $count);
        if ($quotaErr !== null) {
            $this->error_qz($quotaErr);
        }
    }

    /**
     * A6 变更前策略：仅套餐/全局上限（不查实例配额，实例已存在）。
     */
    protected function enforceUpdatePolicy(array $requested, $product = null)
    {
        $cred = $this->currentCredential;
        $site = is_array(config('web')) ? config('web') : [];
        $err = CloudPolicy::enforceResourcePolicy($cred, $requested, $product, $site);
        if ($err !== null) {
            $this->error_qz($err);
        }
    }

    /**
     * F2：续费/开通到期时间校验（防免续费无限延长）。失败即终止。
     *
     * @param mixed $value 到期时间
     */
    protected function assertValidEndTime($value)
    {
        $site = is_array(config('web')) ? config('web') : [];
        $maxDays = isset($site['cloud_max_extend_days']) ? (int)$site['cloud_max_extend_days'] : null;
        $err = CloudPolicy::validateEndTime($value, $maxDays);
        if ($err !== null) {
            $this->error_qz($err);
        }
    }

    /**
     * P2-11：续期/升级必须绑定并原子消费「已支付订单」授权。
     *
     * 返回授权行；无 order_id / 授权无效 / 越权 / 过期 / 重放 / 并发竞争失败
     * 一律返回 null（调用方必须因此拒绝变更，且不改变 end_time/product/
     * resources）。调用方应把 [consumeOrderAuthorization] 与主机状态变更放入
     * **同一 DB 事务**：变更失败时授权消费一并回滚（原子、幂等）。
     *
     * @param string $kind 'renew' | 'update' | 'elastic_update'
     * @param string $hostId 目标主机 ID
     * @return array|null
     */
    protected function consumeOrderAuthorization($kind, $hostId)
    {
        $orderId = trim((string)$this->request->param('order_id', ''));
        if ($orderId === '') {
            return null;
        }
        return \app\common\service\OrderAuthorizationService::consume(
            $orderId,
            $kind,
            (string)$hostId,
            $this->currentCredential
        );
    }

    /**
     * A6：在「同一事务 + 命名应用锁」内原子 reserve/create/commit 云主机，
     * 消除「先 COUNT 后 create」的实例配额 TOCTOU（并发可越限）。
     *
     * 并发语义（与 tests/CloudPolicyTest.php 的 N-way 并发用例同一算法）：
     *   1. 获取租户级命名锁（MySQL GET_LOCK / 测试中的 flock）——串行化同一
     *      租户的配额检查与创建；
     *   2. 在锁内统计现有实例数并执行配额校验；
     *   3. 创建（reserve 计入 quota）；
     *   4. 提交并释放锁；任何失败路径 rollback（释放配额）。
     *
     * $param 为 Ecs::createVps 参数。配额/失败统一走 $onError（error_qz 终止）。
     */
    protected function createHostAtomically(array $param, $onError)
    {
        $cred = $this->currentCredential;
        $userId = isset($cred['user_id']) ? (int)$cred['user_id'] : 0;
        $lockName = 'xiaohei_qz_quota_' . $userId;
        $acquired = false;
        try {
            Db::startTrans();
            $rows = Db::query("SELECT GET_LOCK('" . addslashes($lockName) . "', 10) AS ok");
            $acquired = !empty($rows) && (int)$rows[0]['ok'] === 1;
            if (!$acquired) {
                Db::rollback();
                $onError('配额锁获取失败，请稍后再试');
                return null;
            }
            $count = (new HostVps())->where(['user_id' => $userId, 'is_delete' => 0])->count();
            $quotaErr = CloudPolicy::enforceInstanceQuota($cred, $count);
            if ($quotaErr !== null) {
                Db::query("SELECT RELEASE_LOCK('" . addslashes($lockName) . "')");
                Db::rollback();
                $onError($quotaErr);
                return null;
            }
            $result = Ecs::createVps($param);
            Db::query("SELECT RELEASE_LOCK('" . addslashes($lockName) . "')");
            Db::commit();
            return $result;
        } catch (\Exception $e) {
            try {
                if ($acquired) {
                    Db::query("SELECT RELEASE_LOCK('" . addslashes($lockName) . "')");
                }
            } catch (\Throwable $ignored) {
            }
            Db::rollback();
            throw $e;
        }
    }

    public function test(){
        echo "hello";
    }

    public function product(){
        $productModel = new Product();
        $product = $productModel->select();
        if(!empty($product)){
            $product = $product->toArray();
        }else{
            $this->error_qz('请先完善轻舟后台配置');
        }
        $product = array_column($product,null,'id');
        $this->success_qz('succ',$product);
    }
    /**
 * 获取所有 VPS 的简单信息列表（含 remark）
 */
/**
 * 获取所有 VPS 的简单信息列表（含分页和备注搜索）
 *
 * 请求参数（均为 GET 或 POST）：
 * - limit        每页条数，默认为 16
 * - pages        页码（从 1 开始），默认为 1
 * - search_tag   按 remark 字段模糊搜索
 */
public function hostList()
{
    // 1. 获取请求参数，并设置默认值
    $limit    = $this->request->param('limit/d', 16);
    $page     = $this->request->param('pages/d', 1);
    $search   = $this->request->param('search_tag/s', '');

    // 2. 构造查询：只选取 id、host_name、ip、remark
    $query = (new HostVps())
        ->field('id,host_name,ip,remark');

    // A6/F7：凭据绑定 user_id 时只能看到自己名下的主机（跨租户枚举防护）。
    $scope = CloudPolicy::hostListScope($this->currentCredential);
    if ($scope !== null) {
        $query->where($scope);
    }

    // 3. 如果提供了 search_tag，按 remark 做模糊查询
if ($search !== '') {
    $query->where(function ($q) use ($search) {
        $q->where('remark', 'like', "%{$search}%")
          ->whereOr('host_name', 'like', "%{$search}%");
    });
}
    // 4. 分页查询
    $hosts = $query
        ->page($page, $limit)
        ->select()
        ->toArray();

    // 5. 返回结果
    $this->success_qz('succ', $hosts);
}

    public function line(){
        $lineModel = new ServersLine(); //线路
        $line = $lineModel->where('state=1')->select();
        if(!empty($line)){
            $line = $line->toArray();
        }else{
            $this->error_qz('请先完善轻舟后台配置');
        }
        $lineImageMap = [];
        $lineImageRows = (new ServersImageLine())->field('line_id,config_id')->select();
        if (!empty($lineImageRows)) {
            $lineImageRows = $lineImageRows->toArray();
            foreach ($lineImageRows as $row) {
                $lid = isset($row['line_id']) ? (int)$row['line_id'] : 0;
                $cid = isset($row['config_id']) ? (int)$row['config_id'] : 0;
                if ($lid <= 0 || $cid <= 0) {
                    continue;
                }
                if (!isset($lineImageMap[$lid])) {
                    $lineImageMap[$lid] = [];
                }
                $lineImageMap[$lid][] = $cid;
            }
            foreach ($lineImageMap as $lid => $ids) {
                $lineImageMap[$lid] = array_values(array_unique($ids));
            }
        }

        foreach ($line as &$item) {
            // 第三方线路 apikey 属于凭据，不随线路列表返回
            if (isset($item['line_apikey'])) {
                unset($item['line_apikey']);
            }
            $ids = [];
            $csv = isset($item['line_image_use']) ? trim((string)$item['line_image_use']) : '';
            if ($csv !== '') {
                $ids = array_map('intval', explode(',', $csv));
                $ids = array_values(array_unique(array_filter($ids)));
            }
            if (empty($ids)) {
                $lid = isset($item['id']) ? (int)$item['id'] : 0;
                if ($lid > 0 && isset($lineImageMap[$lid])) {
                    $ids = $lineImageMap[$lid];
                }
            }
            $item['image_ids'] = $ids;
        }
        unset($item);

        $this->success_qz('succ',$line);
    }

    protected function getNodeidForCloud($line_id){
        if (class_exists('\\qzcloud\\Kvm') && !method_exists('\\qzcloud\\Kvm', 'getMemoryAndDisk')) {
            $nodeModel = new ServersNode();
            $hostVpsModel = new HostVps();
            $node_list = $nodeModel->where(['line_id' => $line_id, 'state' => 1])->order('weight desc')->select();
            $ct = $hostVpsModel->field("count(*) ct,node_id")->group('node_id')->where('state!=11')->select();
            $ct_ = [];
            if(!empty($ct)){
                $ct =$ct->toArray();
                $ct_ =array_column($ct,'ct','node_id');
            }
            foreach ($node_list as $v){
                if(isset($ct_[$v['id']])&&$v['max_vm_number']<=$ct_[$v['id']]){
                    continue;
                }
                return $v;
            }
            return null;
        }

        return Ecs::getNodeid($line_id);
    }

    public function openhost(){
        $productid = $this->request->param("productid");
        $templates_id = $this->request->param("templates_id");
        $sys_pwd = $this->request->param("sys_pwd");
        $vnc_password =  $this->request->param("vnc_pwd");
        $expire_time=  $this->request->param("expire_time");
        $line_id=  $this->request->param("line_id");

        try{
            // F2：开通到期时间必须在未来且不超过最大可续窗口。
            $this->assertValidEndTime($expire_time);
            $productModel = new Product();
            if(!$productid){
                $this->error_qz("product_id 错误");
            }
            $product = $productModel->where(['id'=>$productid])->find();
            if(!$product){
                $this->error_qz("product does not exist");
            }
            // A6：只允许开通可售套餐，并做实例配额校验。资源来自套餐
            // (host_cpu/host_ram/host_data/...) 映射进资源字段，杜绝
            // 「openhost 传空 requested 绕过套餐」。
            $productArr = $product->toArray();
            if (isset($productArr['isshow']) && (int)$productArr['isshow'] !== 1) {
                $this->error_qz("product is not purchasable");
            }
            $this->enforceCreatePolicy(CloudPolicy::buildRequestedFromProduct($productArr), $productArr);

            //拼装主机列表参数
            $lineModel = new ServersLine(); //线路
            $nodeModel = new ServersNode();//物理节点信息
            $areaModel = new ServersArea();//地区信息
            $line = $lineModel->where(['id' => $line_id])->find();
            if(!$line || (int)$line['state'] !== 1){
                $this->error_qz("ret=line not available");
            }
            if($line['line_type']==2){
                $this->error_qz("ret=暂时不支持开通上级代理商产品");
            }
            // $node = $nodeModel->where("line_id = '" . $line_id . "' and vm_number<max_vm_number")->order('weight desc')->find();

            $hostVpsModel = new HostVps();
            //and vm_number<max_vm_number
            $node = $this->getNodeidForCloud($line_id);
            $area = $areaModel->where(['id' => $line['area_id']])->find();
            if (!$node) {
                $this->error_qz("ret=No nodes available");
            }
            $param = [];

            $param['product_id'] = $product['id'];
            $param['product_name'] = $product['product_name'];
            // A6：归属来自凭据绑定用户，而不是调用方任意指定。
            $param['user_id'] = isset($this->currentCredential['user_id'])
                ? (int)$this->currentCredential['user_id'] : 0;
            $param['orderid'] = 0;
            $param['area_name'] = $area['area_name'];
            $param['area_id'] = $area['id'];
            $param['other_info'] ="";
            $param['line_name'] = $line['line_name'];
            $param['line_id'] = $line['id'];
            $param['node_name'] = $node['node_name'];
            $param['vlanid1'] = $node['vlan_id1'];
            $param['vlanid2'] = $node['vlan_id2'];
            $param['node_id'] = $node['id'];
            $param['virtual_type'] = $node['virtual_type'];
            $param['from_type'] = 1; //自生产
            $param['buy_time'] = date("Y-m-d H:i:s");
            $param['end_time'] = $expire_time;

            //显卡
            $param['gpu_capacity'] = $product['gpu_capacity'];
            $param['resolution'] = $product['gpu_resolution'];
            //裸金
            $param['metal'] = $product['metal']==1?2:1;

            $param['ipnum'] = $product['ipnum'];
            //$param['virtual_type'] = $virtual_type=='h'?'hyper-v':'kvm';

            $param['cpu'] = $product['host_cpu'];
            $param['memory'] = $product['host_ram'];
            $param['hard_disks'] = $product['host_data'];
            $param['bandwidth'] = $product['bandwidth'];
            $param['os_name'] = $templates_id ;
            $param['os_password'] =$sys_pwd;
            $param['panel_password'] = $vnc_password;
            $param['vnc_password'] = $vnc_password;
            $param['memory_dynamic'] = $node['memory_dynamic'];
            $param['ram_start'] = $node['memory_dynamic']==1?$node['ram_start']:0;
            $param['cpu_limit'] = $node['cpu_limit'];
            $param['state'] = 1;//创建中
            $param['is_agent'] =  0;
            //$param['os_disk_path'] = $node['os_disk_path'];
            $param['os_disk_maxiops'] = $node['os_iops_max'];
            $param['data_disk_path'] = $node['data_path'];
            $param['data_disk_maxiops'] = $node['data_iops_max'];
            $param['is_nat'] = $product['is_nat'];
            if ($product['is_nat'] == 1) { //挂机宝订单
                $param['port_num'] = $product['nat_port_num'];
                $param['domain_num'] = $product['nat_domain_num'];
            }
            $param['snapshot_num'] = $product['snapshot_num'];
            $param['backup_num'] = $product['backup_num'];
            $param['max_reinstall_num'] = $line['reinstall_num'];
            $param['remark'] = 'whmcs';
            // A6：同一事务 + 租户锁内配额校验/创建/提交（失败释放）。
            $result = $this->createHostAtomically($param, function ($msg) {
                $this->error_qz($msg);
            });
            $this->success_qz('succ',$result);
        }catch (Exception $e){
            $this->error_qz($e->getMessage());
        }


    }

    public function renew($host_id,$nextduedate){
        $hostinfo = $this->loadOwnedHost($host_id);
        // F2：续费到期时间必须在未来且不超过最大可续窗口（防免续费无限延长）。
        $this->assertValidEndTime($nextduedate);
        // P2-11：续期不能仅凭 order 作用域免费延长——必须绑定并原子消费
        // 已支付订单授权；消费与 end_time 变更在同一事务内，失败不改 end_time。
        Db::startTrans();
        try {
            $auth = $this->consumeOrderAuthorization('renew', $hostinfo['id']);
            if ($auth === null) {
                Db::rollback();
                $this->error_qz('缺少有效的已支付订单授权（renew 需要 order_id）');
            }
            $hostinfo->save(['end_time'=>$nextduedate]);
            Db::commit();
        } catch (\Exception $e) {
            Db::rollback();
            $this->error_qz($e->getMessage());
        }
        Ecs::unlockHost(['hostid'=>$host_id]);
        $this->success_qz('succ','');
    }

    public function create_host(){
        $post = $this->request->param();

        // A6：服务端必填校验（openapi 契约中的 required 字段），缺失即拒绝。
        foreach (['line_id', 'os', 'cpu', 'memory', 'hard_disks', 'bandwidth', 'expire_time'] as $f) {
            if (!isset($post[$f]) || $post[$f] === '' || $post[$f] === null) {
                $this->error_qz("缺少必填参数 {$f}");
            }
        }
        // F2：创建到期时间必须在未来且不超过最大可续窗口。
        $this->assertValidEndTime($post['expire_time']);

        $lineModel = new ServersLine(); //线路
        $nodeModel = new ServersNode();//物理节点信息


        $line = $lineModel->where(['id' => $post['line_id']])->find();
        if(!$line || (int)$line['state'] !== 1){
            $this->error_qz('line not available');
        }
        if(isset($post['nodes_id'])){
            $node = $nodeModel->where(['id'=>$post['nodes_id']])->find();
        }else{
            $node = $this->getNodeidForCloud($post['line_id']);
            if (!$node) {
                return  $this->json('No nodes available',-1);
            }
        }

        // A6：套餐/全局上限 + 实例配额。指定 product_id 时按其套餐校验。
        $product = null;
        if (!empty($post['product_id'])) {
            $productRow = (new Product())->where(['id' => $post['product_id']])->find();
            if (!$productRow) {
                $this->error_qz('product does not exist');
            }
            $product = $productRow->toArray();
        }
        $this->enforceCreatePolicy([
            'cpu'       => isset($post['cpu']) ? $post['cpu'] : null,
            'memory'    => isset($post['memory']) ? $post['memory'] : null,
            'hard_disks'=> isset($post['hard_disks']) ? $post['hard_disks'] : null,
            'bandwidth' => isset($post['bandwidth']) ? $post['bandwidth'] : null,
            'ipnum'     => isset($post['ipnum']) ? $post['ipnum'] : null,
            'port_num'  => isset($post['port_num']) ? $post['port_num'] : null,
            'snapshot'  => isset($post['snapshot']) ? $post['snapshot'] : null,
            'backups'   => isset($post['backups']) ? $post['backups'] : null,
        ], $product);

        $areaModel = new ServersArea();//地区信息
        $area = $areaModel->where(['id' => $line['area_id']])->find();

        $template_model = new ServersImageConfig();
        $template = $template_model->where(['os_name'=>$post['os']])->find();
        if(empty($template)){
            return  $this->json('image template not found',-1);
        }
        $param = [];
        $param['cpu'] = $post['cpu'];
        // $param['cpu_mode'] = $post['cpu_mode'];
        // A6：归属来自凭据绑定用户；系统凭据才允许沿用调用方 users_id。
        if (isset($this->currentCredential['user_id']) && $this->currentCredential['user_id'] !== null) {
            $param['user_id'] = (int)$this->currentCredential['user_id'];
        } else {
            $param['user_id'] = isset($post['users_id']) ? (int)$post['users_id'] : 0;
        }
        $param['memory'] = $post['memory'];
        $param['hard_disks'] = $post['hard_disks'];
        $param['from_type'] = 1; //自生产
        $param['virtual_type'] = $node['virtual_type'];
        $param['ram_start'] = $node['memory_dynamic']==1?$node['ram_start']:0;
        $param['cpu_limit'] = $node['cpu_limit'];
        $param['state'] = 1;//创建中
        $param['is_agent'] =  0;
        //$param['os_disk_path'] = $node['os_disk_path'];
        $param['os_disk_maxiops'] = $node['os_iops_max'];
        $param['data_disk_path'] = $node['data_path'];
        $param['data_disk_maxiops'] = $node['data_iops_max'];

        //if(isset($post['sys_disk_iops']))$param['os_disk_maxiops'] = $post['sys_disk_iops'];
        //if(isset($post['sys_disk_iops']))$param['os_read'] =$post['sys_disk_read'];
        //if(isset($post['sys_disk_iops']))$param['os_write'] = $post['sys_disk_write'];
        //if(isset($post['sys_disk_iops']))$param['data_disk_maxiops'] = $post['data_disk_iops'];
        //if(isset($post['sys_disk_iops']))$param['data_read'] = $post['data_disk_read'];
        //if(isset($post['sys_disk_iops']))$param['data_write'] = $post['data_disk_write'];
        $param['bandwidth'] = $post['bandwidth'];
        // $param['virtual_type'] = $post['net_in'];
        $param['snapshot_num'] = isset($post['snapshot'])?$post['snapshot']:$line['snapshot_num'];
        $param['backup_num'] = isset($post['backups'])?$post['backups']:$line['backup_num'];
        $param['os_password'] = isset($post['sys_pwd'])?$post['sys_pwd']:'';
        $param['vnc_password'] =isset($post['vnc_pwd'])?$post['vnc_pwd']:'';
        $param['max_reinstall_num'] = (isset($post['max_reinstall_num'])&&!empty($post['max_reinstall_num']))?$post['max_reinstall_num']:$line['reinstall_num'];

        $param['end_time'] = $post['expire_time'];
        $param['ipnum'] = isset($post['ipnum'])?$post['ipnum']:0;
        $param['traffic'] = isset($post['traffic'])?$post['traffic']:0;
        $param['port_num'] =  isset($post['port_num'])?$post['port_num']:$line['port_num'];

        $param['os_name'] = $template->os_name;
        $param['is_nat'] = $param['ipnum']==0?1:0;
        $param['host_name'] = isset($post['host_name'])?$post['host_name']:'';

        $param['area_name'] = $area['area_name'];

        $param['area_id'] = $area['id'];
        $param['line_name'] = $line['line_name'];
        $param['line_id'] = $line['id'];
        $param['node_name'] = $node['node_name'];
        $param['node_id'] = $node['id'];

        $param['buy_time'] = date("Y-m-d H:i:s");

        try {
            // A6：同一事务 + 租户锁内配额校验/创建/提交（失败释放）。
            $result = $this->createHostAtomically($param, function ($msg) {
                $this->error_qz($msg);
            });
            $this->success_qz('succ',$result);
        }catch (\Exception $e){
            $this->error_qz($e->getMessage());
        }

    }

    public function panel(){
        // 受保护端点：apikey 已在 initialize() 校验（需要 panel/read/all 作用域）。
        $host_name = $this->request->post('host_name');
        $password = $this->request->post('panel_password');
        if (empty($host_name) || empty($password)) {
            $this->error_qz('缺少 host_name 或 panel_password');
        }
        $hostModel = new HostVps();
        $hostinfo = $hostModel->where(['host_name'=>$host_name,'panel_password'=>$password])->find();
        // 主机或密码错误时直接拒绝，不把密码回显到任何响应。
        if (!$hostinfo) {
            $this->error_qz('云主机不存在或面板密码错误');
        }
        // 所有权守卫：绑定 user_id 的凭据只能为自有主机签发面板 ticket。
        $this->assertOwnsHost($hostinfo);

        // 签发短时、单次、绑定 host + 调用方的随机 ticket；密码绝不进入 URL。
        $caller = isset($this->currentCredential['id']) ? (string)$this->currentCredential['id'] : 'unknown';
        $ticket = \app\common\service\TicketService::issue('panel', $hostinfo['host_name'], $caller);
        if ($ticket === null) {
            $this->error_qz('面板访问过于频繁，请稍后再试');
        }

        $site = config('web');
        $domian = isset($site['apiurl'])?$site['apiurl']:'';
        $forward_url = url('control/ecs/login');

        if($domian){
            $forward_url = ltrim($forward_url,'http://');
            $forward_url = ltrim($forward_url,'https://');
            $index = strpos($forward_url,'/');
            if(strstr($domian,'http')){
                $forward_url = $domian.substr($forward_url,$index,strlen($forward_url));
            }else{
                $forward_url = 'http://'.$domian.substr($forward_url,$index,strlen($forward_url));
            }
        }
        // 只携带单次 ticket；重定向目标必须为 HTTPS/本机（见 consume 端契约）。
        $this->assertSafeRedirectTarget($forward_url);
        header('Location: '.CloudPolicy::buildPanelRedirect($forward_url, $ticket));
        exit;
    }

    /**
     * 组装主机网络信息。返回 [$network, $remoteIp]。
     */
    protected function buildHostNetwork($hostinfo, $host_id)
    {
        $port = '';
        $remote_ip = $hostinfo->ip;
        //主机ip信息
        $network = [];//ServersIpv4Private
        if ($hostinfo->is_nat == 1) {
            $image_model = new ServersImageConfig();
            $image_info = $image_model->where(['os_name' => $hostinfo->os_name])->find();
            $port_vps_model = new ForwardPortVps();
            $portlist = $port_vps_model->where(['host_id' => $host_id, 'sys' => 2])->column('sport', 'dport');
            if ($image_info->os_type == 1) {
                $port = $portlist['3389'];
            } else {
                $port = $portlist['22'];
            }
            $node_model = new ServersNode();
            $node = $node_model->where(['id' => $hostinfo->node_id])->find();
            $remote_ip = $node->forward_url . ':' . $port;
            $natip_model = new ServersIpv4Nat();
            $natip = $natip_model->where(['ip' => $hostinfo->ip])->find();
            $natip->public_ip = $node->forward_url;
            $network['eth1'] = [$natip];
        } else {
            $ip_model = new ServersIpv4();
            $ip = $ip_model->where(['v_name' => $hostinfo->host_name])->select();
            $ip = $ip->toArray();
            foreach ($ip as $k => $v) {
                if ($v['ip'] == $hostinfo['ip']) {
                    unset($ip[$k]);
                    array_unshift($ip, $v);
                }
            }
            $network['eth1'] = $ip;
        }

        $privateip_model = new ServersIpv4Private();
        $privateip = $privateip_model->where(['v_name' => $hostinfo->host_name])->select();
        $network['eth2'] = $privateip;

        return [$network, $remote_ip];
    }

    public function hostinfo($host_id){
        $hostinfo = $this->loadOwnedHost($host_id);
        list($network, $remote_ip) = $this->buildHostNetwork($hostinfo, $host_id);
        // A3：公开视图剔除节点秘密（os_password/panel_password/vnc_password）。
        $this->success_qz('succ', CloudPolicy::publicHostInfo($hostinfo->toArray(), $network, $remote_ip));
    }

    /**
     * A3 迁移兼容：返回含节点秘密的完整主机信息。
     *
     * 仅拥有 `credentials` 作用域（或显式 action）的凭据可调用；普通
     * `hostinfo` 不再携带秘密字段。自动化集成（lightboat 等）需要取回
     * 面板/系统/VNC 密码时改调本端点。
     */
    public function hostinfo_credentials($host_id){
        $hostinfo = $this->loadOwnedHost($host_id);
        list($network, $remote_ip) = $this->buildHostNetwork($hostinfo, $host_id);
        $hostinfo->network = $network;
        $hostinfo->remote_ip = $remote_ip;
        $this->success_qz('succ', $hostinfo);
    }

    public function start(){
        $hostid= $this->request->param("host_id");
        $this->loadOwnedHost($hostid);
        try{
            Ecs::startHost(['hostid'=>$hostid]);
            $this->success_qz('启动命令执行成功');
        }catch (Exception $e){
            $this->error_qz($e->getMessage());
        }
    }

    public function reboot(){
        $hostid = $this->request->param('host_id');
        $this->loadOwnedHost($hostid);
        try{
            Ecs::restartHost(['hostid'=>$hostid]);
            $this->success_qz('重启命令执行成功');
        }catch (Exception $e){
            $this->error_qz($e->getMessage());
        }
    }

    public function shutdown(){
        $hostid = $this->request->param('host_id');
        $this->loadOwnedHost($hostid);
        try{
            Ecs::closeHost(['hostid'=>$hostid]);
            $this->success_qz('关机命令执行成功');
        }catch (Exception $e){
            $this->error_qz($e->getMessage());
        }
    }

    public function delete(){
        $hostid= $this->request->param("host_id");
        $this->loadOwnedHost($hostid);

        try{
            Ecs::delete_host(['hostid'=>$hostid]);
            $this->success_qz('删除命令执行成功',"");
        }catch (Exception $e){
            $this->error_qz($e->getMessage());
        }
    }

    public function lock(){
        $hostid= $this->request->param("host_id");
        if(empty($hostid)){
            $this->error_qz('id is null');
        }
        $this->loadOwnedHost($hostid);

        try{
            Ecs::lockHost(['hostid'=>$hostid]);
            $this->success_qz('succ');
        }catch (Exception $e){
            $this->error_qz($e->getMessage());
        }
    }

    public function unlock(){
        $hostid= $this->request->param("host_id");
        if(empty($hostid)){
            $this->error_qz('id is null');
        }
        $this->loadOwnedHost($hostid);
        try{
            Ecs::unlockHost(['hostid'=>$hostid]);
            $this->success_qz('succ');
        }catch (Exception $e){
            $this->error_qz($e->getMessage());
        }
    }

    public function update(){
        try{
            $productid= $this->request->param('product_id');
            $host_id= $this->request->param("host_id");
            $productModel = new Product();
            $product = $productModel->where(['id'=>$productid])->find();
            // A6：所有权校验（绑定用户只能变更自己的主机）。
            $hostinfo = $this->loadOwnedHost($host_id);
            if(!$product){
                $this->error_qz("ret=product does not exist");
            }
            // A6：只允许变更到可售套餐。
            if (isset($product['isshow']) && (int)$product['isshow'] !== 1) {
                $this->error_qz("ret=product is not purchasable");
            }
            if($product['is_nat']!=$hostinfo['is_nat']){
                $this->error_qz("ret=套餐之间不能互相转换");
            }

            $param['cpu'] = $product['host_cpu'];
            $param['memory'] = $product['host_ram'];
            $param['hard_disks'] = $product['host_data'];
            $param['bandwidth'] = $product['bandwidth'];
            $param['domain_num'] = $product['nat_domain_num'];
            $param['backup_num'] = $product['backup_num'];
            $param['port_num'] = $product['nat_port_num'];
            $param['snapshot_num'] = $product['snapshot_num'];
            $param['host_name'] = $hostinfo['host_name'];
            // P2-11：升级不能仅凭 order 作用域换高配产品——必须绑定并原子
            // 消费已支付订单授权；消费与资源变更在同一事务内，失败不改 resources。
            Db::startTrans();
            try {
                $auth = $this->consumeOrderAuthorization('update', $hostinfo['id']);
                if ($auth === null) {
                    Db::rollback();
                    $this->error_qz('缺少有效的已支付订单授权（update 需要 order_id）');
                }
                Ecs::updateVps($param,$hostinfo);
                Db::commit();
            } catch (\Exception $e) {
                Db::rollback();
                throw $e;
            }
        }catch (Exception $e){
            $this->error_qz($e->getMessage());
        }
        $this->success_qz('ret=ok');
    }


    public function error_qz($msg,$code=0){
        //return json(['code'=>$code,'msg'=>$msg]);
        header('content-type:application/json');
        echo  json_encode(['code'=>$code,'msg'=>$msg]);die;
    }

    public function success_qz($msg,$data='',$code=1){
        header('content-type:application/json');
        echo  json_encode(['code'=>$code,'msg'=>$msg,'data'=>$data]);die;
    }

    public function snapshot_list($host_id){
        $this->loadOwnedHost($host_id);
        $snapshotHostModel = new SnapshotVps();
        $list = $snapshotHostModel->where(['host_id'=>$host_id])->select();//
        $data = [];
        foreach ($list as $k=>$v){
            $data[] = [
                'id'=>$v['id'],
                'virtuals_id'=>$v['host_id'],
                'name'=>$v['name'],
                'state'=>(int)$v['state'],
                'created_at'=>$v['create_time']
            ];
        }
        $this->success_qz('succ',$data);
    }

    /**
     * 快照/备份任务触发后，主动短轮询真实状态并回写主机状态。
     * 说明：底层任务异步执行，这里做有限次数轮询，尽量在操作后及时同步状态。
     */
    protected function syncHostStateAfterSnapshotOrBackup($host_id, $retry = 8, $sleepSeconds = 1){
        $hostModel = new HostVps();
        $host = $hostModel->where(['id' => $host_id])->find();
        if (empty($host)) {
            return;
        }

        // 仅对可实时采集的运行/关机场景执行同步
        if (!in_array((int)$host->state, [2, 3], true)) {
            return;
        }

        $logic = new \app\common\service\Ecs();
        for ($i = 0; $i < $retry; $i++) {
            $result = $logic->getStateHost(['hostid' => $host_id]);
            if (isset($result['code']) && (int)$result['code'] === 200 && isset($result['data']['state'])) {
                $isRunning = strtolower((string)$result['data']['state']) === 'running';
                $targetState = $isRunning ? 2 : 3;
                $hostModel->where(['id' => $host_id])->update([
                    'state' => $targetState,
                    'update_time' => date('Y-m-d H:i:s'),
                ]);
                return;
            }
            sleep($sleepSeconds);
        }
    }

    public function backups_add($host_id){
        $this->loadOwnedHost($host_id);
        $post = $this->request->param();
        $data = Ecs::createBackupHost(['hostid'=>$host_id]);
        if($data['code']==200){
            $this->syncHostStateAfterSnapshotOrBackup($host_id);
            $this->success_qz('succ',$data);
        }else{
            $this->error_qz($data['msg']);
        }
    }

    public function backups_del($host_id,$id){
        $this->loadOwnedHost($host_id);
        $backupVpsModel = new BackupVps();
        $backupVps = $backupVpsModel->find($id);
        if(!$backupVps || (int)$backupVps['host_id'] !== (int)$host_id){
            $this->error_qz('备份不存在');
        }
        $data = Ecs::removeBackupHost(['hostid'=>$host_id,'id'=>$id]);
        if($data['code']==200){
            $this->syncHostStateAfterSnapshotOrBackup($host_id);
            $this->success_qz('succ',$data);
        }else{
            $this->error_qz($data['msg']);
        }
    }

    public function backups_restore($host_id,$id){
        $this->loadOwnedHost($host_id);
        $backupVpsModel = new BackupVps();
        $backupVps = $backupVpsModel->find($id);
        if(!$backupVps || (int)$backupVps['host_id'] !== (int)$host_id){
            $this->error_qz('备份不存在');
        }
        $data = Ecs::restoreBackupHost(['hostid'=>$host_id,'id'=>$id]);
        if($data['code']==200){
            $this->syncHostStateAfterSnapshotOrBackup($host_id);
            $this->success_qz('succ',$data);
        }else{
            $this->error_qz($data['msg']);
        }
    }

    public function backups_list($host_id){
        $this->loadOwnedHost($host_id);
        $backupVpsModel = new BackupVps();
        $list = $backupVpsModel->where(['host_id'=>$host_id])->select();//
        $data = [];
        foreach ($list as $k=>$v){
            $data[] = [
                'id'=>$v['id'],
                'virtuals_id'=>$v['host_id'],
                'name'=>$v['name'],
                'state'=>(int)$v['state'],
                'created_at'=>$v['create_time']
            ];
        }
        $this->success_qz('succ',$data);
    }

    public function mirror_image(){
        $line_id = (int)$this->request->param('line_id', 0);
        $list = [];

        if ($line_id > 0) {
            $lineModel = new ServersLine();
            $lineInfo = $lineModel->where(['id' => $line_id])->find();
            if (empty($lineInfo)) {
                $this->error_qz('line_id 错误');
            }

            $line_image_use = isset($lineInfo['line_image_use']) ? trim((string)$lineInfo['line_image_use']) : '';
            if ($line_image_use !== '') {
                $image_ids = array_map('intval', explode(',', $line_image_use));
                $image_ids = array_values(array_unique(array_filter($image_ids)));
                if (!empty($image_ids)) {
                    $image_model = new ServersImageConfig();
                    $imageListObj = $image_model->whereIn('id', $image_ids)->order('sort asc,id asc')->select();
                    if ($imageListObj && !$imageListObj->isEmpty()) {
                        $list = $imageListObj->toArray();
                    }
                }
            }

            if (empty($list)) {
                $lineImageModel = new ServersImageLine();
                $lineImageObj = $lineImageModel->where(['line_id' => $line_id])->order('sort asc,id asc')->select();
                if ($lineImageObj && !$lineImageObj->isEmpty()) {
                    $list = $lineImageObj->toArray();
                }
            }

            // 线路未配置镜像时回退全量，避免同步把镜像关系列表写空
            if (empty($list)) {
                $image_model = new ServersImageConfig();
                $list = $image_model->order('sort asc,id asc')->select()->toArray();
            }
        } else {
            $image_model = new ServersImageConfig();
            $list = $image_model->order('sort asc,id asc')->select()->toArray();
        }

        $data = [];
        foreach ($list as $k=>$v){
            $image_id = (isset($v['config_id']) && $v['config_id']) ? $v['config_id'] : $v['id'];
            $data[] = [
                'id'=>$image_id,
                'image_id'=>$image_id,
                'name'=>$v['os_name'],
                'type'=>isset($v['os_type_name']) ? $v['os_type_name'] : (isset(ServersImageConfig::$os_type[$v['os_type']]) ? ServersImageConfig::$os_type[$v['os_type']] : '未知'),
                'desc'=>$v['remark'],
            ];
        }
        $this->success_qz('succ',$data);
    }

    public function nat_acl_list($host_id){
        $this->loadOwnedHost($host_id);
        $forwardPortModel= new ForwardPortVps();
        $list = $forwardPortModel->where(['host_id'=>$host_id])->order('id desc')->select();

        $this->success_qz('succ',$list);
    }

    public function cdrom($host_id=0){
        if ($host_id > 0) {
            $this->loadOwnedHost($host_id);
        }
        $this->success_qz('succ',[]);
    }

    public function security_acl_list($host_id){
        $this->loadOwnedHost($host_id);
        $data = (new FirewallVps())->where(['host_id'=>$host_id])->select();
        $this->success_qz('succ',$data);
    }

    //资源监控
    public function monitor($host_id){

        /**
        $key = "ip:{$ip}:monitor_host";
        $limit = 10;
        try{
        $redis= new \Redis();
        $redis->connect(config('REDIS_HOST'),config('REDIS_PORT'));
        $redis->Expire = config('REDIS_EXPIRE');
        }catch (\Exception $e){
        Log::error($e->getMessage());
        exit('redis 连接错误');
        }

        $check = $redis->exists($key);
        if($check){
        $redis->incr($key);  //键值递增
        $count = $redis->get($key);
        if($count > $limit){
        exit('your have too many request');
        }
        }else{
        $redis->incr($key);
        //限制时间为60秒
        $redis->expire($key,60);
        }
         **/
        $this->loadOwnedHost($host_id);
        $logic = new \app\common\service\Ecs();
        $data = $logic->monitorHost(['hostid'=>$host_id]);
        if($data['code']==200){
            $this->success_qz('success',$data['data']);
        }else{
            $this->error_qz($data['msg']);
        }
    }

    //云主机实时运行状态
    public function state_host($host_id){
        $this->loadOwnedHost($host_id);
        $logic = new \app\common\service\Ecs();
        $host = (new HostVps())->where(['id'=>$host_id])->find();
        if(empty($host)){
            $this->error_qz('云主机不存在');
        }

        if(!in_array((int)$host->state,[2,3],true)){
            $label = isset(BaseConst::HOST_STATE[$host->state]) ? BaseConst::HOST_STATE[$host->state] : '';
            $this->success_qz('succ',[(int)$host->state,$label]);
        }

        $data = $logic->getStateHost(['hostid'=>$host_id]);
        if($data['code']!=200){
            $msg = isset($data['msg']) ? $data['msg'] : '采集状态失败';
            $this->error_qz($msg);
        }

        $isRunning = strtolower((string)$data['data']['state'])=='running';
        $state = $isRunning?2:3;
        $label = isset(BaseConst::HOST_STATE[$state]) ? BaseConst::HOST_STATE[$state] : '';
        $this->success_qz('succ',[$state,$label]);
    }

    public function reset_os($host_id,$template_id){
        // 回归守卫：新系统密码只从受保护 POST body 接收，绝不出现在 URL 路径/query。
        $password = $this->request->post('password', '');
        if (!is_string($password)) {
            $this->error_qz('password 必须为字符串');
        }
        try{
            $hostinfo = $this->loadOwnedHost($host_id);
            $image = new ServersImageConfig();
            $image_config = $image->where('id', $template_id)->whereOr('os_name', $template_id)->find();
            if(empty($image_config)){
                $this->error_qz('镜像不存在');
            }
            $param['os_id'] =$image_config['id'];
            $param['hostid'] = $host_id;
            $param['password'] = ($password==='')?$hostinfo['os_password']:$password;
            $result = Ecs::reinstallTask($param);
            if($result['code']==200){
                $this->success_qz('已放入重装列队中',[]);
            }else{
                $this->error_qz($result['msg']);
            }
        }catch (Exception $e){
            $this->error_qz($e->getMessage());
        }
    }

    public function snapshot_add($host_id){
        $this->loadOwnedHost($host_id);
        $data = Ecs::createSnapshotHost(['hostid'=>$host_id]);
        if($data['code']==200){
            $this->syncHostStateAfterSnapshotOrBackup($host_id);
            $this->success_qz('succ',$data);
        }else{
            $this->error_qz($data['msg']);
        }
    }

    public function snapshot_del($host_id,$id){
        $this->loadOwnedHost($host_id);
        $snapshotHostModel = new SnapshotVps();
        $snapshot = $snapshotHostModel->find($id);
        if(!$snapshot || (int)$snapshot['host_id'] !== (int)$host_id){
            $this->error_qz('快照不存在');
        }
        $data = Ecs::removeSnapshotHost(['hostid'=>$host_id,'id'=>$id]);
        if($data['code']==200){
            $this->syncHostStateAfterSnapshotOrBackup($host_id);
            $this->success_qz('succ',$data);
        }else{
            $this->error_qz($data['msg']);
        }
    }

    public function snapshot_restore($host_id,$id){
        $this->loadOwnedHost($host_id);
        $snapshotHostModel = new SnapshotVps();
        $snapshot = $snapshotHostModel->find($id);
        if(!$snapshot || (int)$snapshot['host_id'] !== (int)$host_id){
            $this->error_qz('快照不存在');
        }
        $data = Ecs::restoreSnapshotHost(['hostid'=>$host_id,'id'=>$id]);
        if($data['code']==200){
            $this->syncHostStateAfterSnapshotOrBackup($host_id);
            $this->success_qz('succ',$data);
        }else{
            $this->error_qz($data['msg']);
        }
    }

    public function security_acl_add(){
        $post = $this->request->param();

        if(empty($post['hostid'])&&!empty($post['host_id'])){
            $post['hostid'] = $post['host_id'];
        }
        if(!isset($post['port'])&&isset($post['start_port'])){
            $post['port'] = $post['start_port'];
        }
        if(!isset($post['ip'])&&isset($post['start_ip'])){
            $post['ip'] = $post['start_ip'];
        }

        if(!isset($post['direction'])&&isset($post['orientation'])){
            $post['direction'] = $post['orientation'];
        }
        if(isset($post['direction'])){
            $direction = strtolower((string)$post['direction']);
            if(in_array($direction,['1','in','ingress'],true)){
                $post['direction'] = 'in';
            }elseif(in_array($direction,['2','out','egress'],true)){
                $post['direction'] = 'out';
            }
        }

        if(isset($post['protocol'])){
            $protocol = strtoupper((string)$post['protocol']);
            $post['protocol'] = in_array($protocol,['TCP','UDP','ICMP','ANY','ALL'],true) ? $protocol : 'ANY';
            if($post['protocol']=='ALL'){
                $post['protocol'] = 'ANY';
            }
        }else{
            $post['protocol'] = 'ANY';
        }

        if(isset($post['method'])){
            $method = strtolower((string)$post['method']);
            if(in_array($method,['accept','allow'],true)){
                $post['method'] = 'accept';
            }elseif(in_array($method,['drop','deny','reject'],true)){
                $post['method'] = 'drop';
            }
        }elseif(isset($post['tactics'])){
            $post['method'] = ((string)$post['tactics']==='1') ? 'drop' : 'accept';
        }else{
            $post['method'] = 'accept';
        }

        if(!isset($post['port'])||$post['port']===''){
            $post['port'] = -1;
        }
        if(!isset($post['ip'])||$post['ip']===''){
            $post['ip'] = '0.0.0.0';
        }

        // 防止底层 Hyper-V 解析空字符串时报“输入字符串的格式不正确”
        if(!isset($post['priority'])||$post['priority']===''){
            $post['priority'] = 100;
        }

        if(empty($post['hostid'])||empty($post['direction'])||empty($post['method'])){
            $this->error_qz('参数不完整，至少需要 hostid/host_id, direction, method');
        }
        $this->loadOwnedHost($post['hostid']);

        $data = Ecs::add_firewall_host($post);
        if($data['code']==200){
            $this->success_qz('succ',$data);
        }else{
            $this->error_qz($data['msg']);
        }
    }

    public function security_acl_del($host_id,$id){
        $this->loadOwnedHost($host_id);
        $data =  Ecs::remove_firewall_host(['hostid'=>$host_id,'id'=>$id]) ;
        if($data['code']==200){
            $this->success_qz('succ',$data);
        }else{
            $this->error_qz($data['msg']);
        }
    }
    
    
    
        public function findport(){
        $keywords = $this->request->param('keywords');
        $hostid = $this->request->param('hostid');
        if (!empty($hostid)) {
            $this->loadOwnedHost($hostid);
        }
        $data = \app\common\service\Ecs::find_port(['keywords'=>$keywords,'hostid'=>$hostid]);
        //查找端口
        return json(['code'=>0,'content'=>$data,'type'=>'success']);
    }

    //添加一个端口
public function add_port_host(){
        $logic = new \app\common\service\Ecs();
        $param = $this->request->param();
        if (empty($param['host_id']) && !empty($param['hostid'])) {
            $param['host_id'] = $param['hostid'];
        }
        if (empty($param['host_id'])) {
            $this->error_qz('缺少 host_id');
        }
        $this->loadOwnedHost($param['host_id']);
        if(!empty($param['sport'])){
            $info = $logic->find_port(['keywords'=>$param['sport'],'hostid'=>$param['host_id'],'like'=>2]);
            if(empty($info)){
                $this->error_qz('抱歉您添加的公网端口被占用，info=' . json_encode($info, JSON_UNESCAPED_UNICODE));
            }
        }else{
            $info = $logic->find_port(['hostid'=>$param['host_id'],'keywords'=>'']);
            if(empty($info)){
                $this->error_qz('没有可用端口');
            }
            $param['sport'] = $info[0];
        }
        $param['hostid']= $param['host_id'];
        $data = $logic->add_forward_port($param);
        if($data['code']==200){
            $this->success_qz('添加端口成功','');
        }else{
            $this->error_qz($data['msg']);
        }
    }


    //删除端口
    public function remove_port_host(){
        $logic = new \app\common\service\Ecs();
        $param = $this->request->param();
        // 目标主机必填：缺 host_id 直接 fail-closed，绝不跳过所有权校验删除映射。
        $hostId = !empty($param['host_id']) ? $param['host_id'] : (isset($param['hostid']) ? $param['hostid'] : 0);
        if (empty($hostId)) {
            $this->error_qz('缺少 host_id');
        }
        // 所有权校验：凭据绑定 user_id 时只能删除自有主机的端口映射。
        $this->loadOwnedHost($hostId);
        // 映射归属校验：端口映射 id 必须属于目标主机，跨主机/跨租户删除 fail-closed。
        $portId = isset($param['id']) ? $param['id'] : (isset($param['port_id']) ? $param['port_id'] : 0);
        $mapping = (new ForwardPortVps())->where(['id' => $portId, 'host_id' => $hostId])->find();
        $portErr = CloudPolicy::validatePortRemoval($hostId, $portId, $mapping ? $mapping->toArray() : null);
        if ($portErr !== null) {
            $this->error_qz($portErr);
        }
        $param['hostid'] = $hostId;
        $data = $logic->remove_forward_port($param);
        if($data['code']==200){
            $this->success_qz('删除端口成功','');
        }else{
            $this->error_qz($data['msg']);
        }
    }

    public function reset_password($host_id){
        $this->loadOwnedHost($host_id);
        $password = $this->request->post('password');
        $result = Ecs::update_password_host(['hostid'=>$host_id,'password'=>$password]);
        if($result['code']==200){
            $this->success_qz('操作成功','');

        }else{
            $this->error_qz($result['msg']);
        }

    }

    public function vnc_view($host_id){
        $hostinfo = $this->loadOwnedHost($host_id);
        // 回归守卫：vnc_password 不得进入重定向 URL/query。签发短时单次
        // ticket，客户端用 ticket 到同源 consume 端点换取服务端会话。
        $caller = isset($this->currentCredential['id']) ? (string)$this->currentCredential['id'] : 'unknown';
        $ticket = \app\common\service\TicketService::issue('vnc', (string)$hostinfo['id'], $caller);
        if ($ticket === null) {
            $this->error_qz('VNC 访问过于频繁，请稍后再试');
        }
        $this->success_qz('succ', [
            'ticket' => $ticket,
            'consume_url' => url('control/ecs/vnc') . '?ticket=' . urlencode($ticket),
        ]);
    }

    //自由选配升级
public function elastic_update(){
    try {
        $post = $this->request->param();

        // 必须传 host_id
        if (empty($post['host_id'])) {
            $this->error_qz('缺少 host_id');
        }

        // A6：所有权 + 套餐/全局上限校验（变更不允许超出宿主所属套餐）。
        $hostinfo = $this->loadOwnedHost($post['host_id']);
        $product = null;
        if (!empty($hostinfo['product_id'])) {
            $productRow = (new Product())->where(['id' => $hostinfo['product_id']])->find();
            if ($productRow) {
                $product = $productRow->toArray();
            }
        }
        $this->enforceUpdatePolicy([
            'cpu'        => isset($post['cpu']) ? $post['cpu'] : null,
            'memory'     => isset($post['memory']) ? $post['memory'] : null,
            'hard_disks' => isset($post['hard_disks']) ? $post['hard_disks'] : null,
            'bandwidth'  => isset($post['bandwidth']) ? $post['bandwidth'] : null,
            'backups'    => isset($post['backups']) ? $post['backups'] : null,
            'port_num'   => isset($post['port_num']) ? $post['port_num'] : null,
            'snapshot'   => isset($post['snapshot']) ? $post['snapshot'] : null,
        ], $product);

        // 构造参数，只保证 id 是必须的，其它字段如果不传或为空就不改
        $param = [
            'id'        => $post['host_id'],
            'host_name' => $hostinfo['host_name'],  // 始终带上
        ];

        if (isset($post['cpu']) && $post['cpu'] !== '') {
            $param['cpu'] = $post['cpu'];
        }
        if (isset($post['memory']) && $post['memory'] !== '') {
            $param['memory'] = $post['memory'];
        }
        if (isset($post['hard_disks']) && $post['hard_disks'] !== '') {
            $param['hard_disks'] = $post['hard_disks'];
        }
        if (isset($post['bandwidth']) && $post['bandwidth'] !== '') {
            $param['bandwidth'] = $post['bandwidth'];
        }
        if (isset($post['backups']) && $post['backups'] !== '') {
            $param['backup_num'] = $post['backups'];
        }
        if (isset($post['port_num']) && $post['port_num'] !== '') {
            $param['port_num'] = $post['port_num'];
        }
        if (isset($post['snapshot']) && $post['snapshot'] !== '') {
            $param['snapshot_num'] = $post['snapshot'];
        }
        if (isset($post['ip_num']) && $post['ip_num'] !== '') {
            $param['ip_num'] = $post['ip_num'];  // 可选，不传则不修改
        }

        // P2-11：自由选配升级不能仅凭 order 作用域免费加配——必须绑定并原子
        // 消费已支付订单授权；消费与资源变更在同一事务内，失败不改 resources。
        Db::startTrans();
        try {
            $auth = $this->consumeOrderAuthorization('elastic_update', $hostinfo['id']);
            if ($auth === null) {
                Db::rollback();
                $this->error_qz('缺少有效的已支付订单授权（elastic_update 需要 order_id）');
            }
            Ecs::updateVps($param, $hostinfo);
            Db::commit();
        } catch (\Exception $e) {
            Db::rollback();
            throw $e;
        }
        $this->success_qz('升级成功');
    } catch (\Exception $e) {
        $this->error_qz($e->getMessage());
    }
}


    public function hostinfo_byname($host_name){

        $hostModel = new HostVps();
        $hostinfo = $hostModel->where(['host_name'=>$host_name])->find();
        if(empty($hostinfo)){
            $this->error_qz('云主机不存在');
        }
        // A6：所有权校验。
        $this->assertOwnsHost($hostinfo);
        // A3：公开视图剔除节点秘密。
        list($network, $remote_ip) = $this->buildHostNetwork($hostinfo, $hostinfo['id']);
        $this->success_qz('succ', CloudPolicy::publicHostInfo($hostinfo->toArray(), $network, $remote_ip));
    }
}

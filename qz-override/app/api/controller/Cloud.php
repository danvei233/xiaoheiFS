<?php

namespace app\api\controller;

use app\ApiController;
use app\common\model\HostVps;
use app\common\model\Product;
use app\common\model\ServersArea;
use app\common\model\ServersLine;
use app\common\model\ServersNode;
use app\common\model\ServersImageConfig;
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

/**
 * 首页接口
 */
class Cloud extends ApiController
{
    protected $noNeedLogin = ['*'];
    protected $noNeedRight = ['*'];

    protected function initialize()
    {
        $apikey= $this->request->header("apikey");
        $site = config('web');
        $localapikey = $site['apikey'];
        if($localapikey!=$apikey&&$this->request->action()!='panel'){
             $this->error_qz("apikey 错误");
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
        $this->success_qz('succ',$line);
    }

    protected function getNodeidForCloud($line_id){
        if (class_exists('\\qzcloud\\Kvm') && !method_exists('\\qzcloud\\Kvm', 'getMemoryAndDisk')) {
            $nodeModel = new ServersNode();
            $hostVpsModel = new HostVps();
            $node_list = $nodeModel->where("line_id = '" . $line_id . "' and state=1 ")->order('weight desc')->select();
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
            $productModel = new Product();
            if(!$productid){
                $this->error_qz("product_id 错误");
            }
            $product = $productModel->where(['id'=>$productid])->find();
            if(!$product){
                $this->error_qz("product does not exist");
            }

            //拼装主机列表参数
            $lineModel = new ServersLine(); //线路
            $nodeModel = new ServersNode();//物理节点信息
            $areaModel = new ServersArea();//地区信息
            $line = $lineModel->where(['id' => $line_id])->find();
            if($line['line_type']==2){
                $this->error_qz("ret=暂时不支持开通上级代理商产品");
            }
            // $node = $nodeModel->where("line_id = '" . $line_id . "' and vm_number<max_vm_number")->order('weight desc')->find();

            $hostVpsModel = new HostVps();
            //and vm_number<max_vm_number
            $node = $this->getNodeidForCloud($line_id);
            if (!$node) {
                $this->wirteline("ret=No nodes available");
            }

            $area = $areaModel->where(['id' => $line['area_id']])->find();
            if (!$node) {
                $this->error_qz("ret=No nodes available");
            }
            $param = [];

            $param['product_id'] = $product['id'];
            $param['product_name'] = $product['product_name'];
            $param['user_id'] = 0;
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
            $result = Ecs::CreateVps($param);
            $this->success_qz('succ',$result);
        }catch (Exception $e){
            $this->error_qz($e->getMessage());
        }


    }

    public function renew($host_id,$nextduedate){
        $hostModel = new HostVps();
        $hostinfo = $hostModel->where(['id'=>$host_id])->find();
        if(!$hostinfo){
            $this->error_qz("ret=err host not found");
        }
        $hostinfo->save(['end_time'=>$nextduedate]);
        Ecs::unlockHost(['hostid'=>$host_id]);
        $this->success_qz('succ','');
    }

    public function create_host(){
        $post = $this->request->param();

        $lineModel = new ServersLine(); //线路
        $nodeModel = new ServersNode();//物理节点信息


        $line = $lineModel->where(['id' => $post['line_id']])->find();
        if(isset($post['nodes_id'])){
            $node = $nodeModel->where(['id'=>$post['nodes_id']])->find();
        }else{
            $node = $this->getNodeidForCloud($post['line_id']);
            if (!$node) {
                return  $this->json('No nodes available',-1);
            }
        }

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
        $param['user_id'] =  isset($post['users_id'])?$post['users_id']:0;
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
            Db::startTrans();
            $result = Ecs::createVps($param);
            Db::commit();

            $this->success_qz('succ',$result);
        }catch (\Exception $e){
            Db::rollback();
            $this->error_qz($e->getMessage());
        }

    }

    public function panel(){
        $host_name = $this->request->param('host_name');
        $password = $this->request->param('panel_password');
        $hostModel = new HostVps();
        $hostinfo = $hostModel->where(['host_name'=>$host_name,'panel_password'=>$password])->find();
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
        header('Location: '.$forward_url.'?host_name='.$hostinfo['host_name'].'&panel_password='.$password);
        exit;
    }

    public function hostinfo($host_id){

        $hostModel = new HostVps();
        $hostinfo = $hostModel->where(['id'=>$host_id])->find();

        $port = '';
        $remote_ip = $hostinfo->ip;
        //主机ip信息
        $network =[];//ServersIpv4Private
        if($hostinfo->is_nat==1){
            $image_model = new ServersImageConfig();
            $image_info = $image_model->where(['os_name'=>$hostinfo->os_name])->find();
            $port_vps_model = new ForwardPortVps();
            $portlist = $port_vps_model->where(['host_id'=>$host_id,'sys'=>2])->column('sport','dport');
            if($image_info->os_type==1){
                $port = $portlist['3389'];
            }else{
                $port =$portlist['22'];
            }
            $node_model = new ServersNode();
            $node = $node_model ->where(['id'=>$hostinfo->node_id])->find();
            $remote_ip = $node->forward_url.':'.$port;
            $natip_model = new ServersIpv4Nat();
            $natip = $natip_model->where(['ip'=>$hostinfo->ip])->find();
            $natip ->public_ip = $node->forward_url;
            //$network
            $network['eth1'] = [$natip];
        }else{
            $ip_model = new ServersIpv4();
            $ip = $ip_model->where(['v_name'=>$hostinfo->host_name])->select();
            $ip = $ip->toArray();
            foreach ($ip as $k=>$v){
                if($v['ip']==$hostinfo['ip']){
                    unset($ip[$k]);
                    array_unshift($ip,$v);
                }
            }
            $network['eth1'] = $ip;
        }



        $privateip_model = new ServersIpv4Private();
        $privateip = $privateip_model->where(['v_name'=>$hostinfo->host_name])->select();
        $network['eth2'] = $privateip;


        $hostinfo->network = $network;
        $hostinfo->remote_ip=$remote_ip;
        $this->success_qz('succ',$hostinfo);

    }

    public function start(){
        $hostid= $this->request->param("host_id");
        try{
            Ecs::startHost(['hostid'=>$hostid]);
            $this->success_qz('启动命令执行成功');
        }catch (Exception $e){
            $this->error_qz($e->getMessage());
        }
    }

    public function reboot(){
        $hostid = $this->request->param('host_id');
        try{
            Ecs::restartHost(['hostid'=>$hostid]);
            $this->success_qz('重启命令执行成功');
        }catch (Exception $e){
            $this->error_qz($e->getMessage());
        }
    }

    public function shutdown(){
        $hostid = $this->request->param('host_id');
        try{
            Ecs::closeHost(['hostid'=>$hostid]);
            $this->success_qz('关机命令执行成功');
        }catch (Exception $e){
            $this->error_qz($e->getMessage());
        }
    }

    public function delete(){
        $hostid= $this->request->param("host_id");
        $hostModel = new HostVps();
        $hostInfo = $hostModel->where(['id'=>$hostid])->find();


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
            $hostModel = new HostVps();
            $productModel = new Product();
            $product = $productModel->where(['id'=>$productid])->find();
            $hostinfo = $hostModel->where(['id'=>$host_id])->find();
            if(!$hostinfo){
                $this->error_qz("ret=云主机不存在");
            }
            if($product['is_nat']!=$hostinfo['is_nat']){
                $this->error_qz("ret=套餐之间不能互相转换");
            }
            if(!$product){
                $this->error_qz("ret=product does not exist");
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
            Ecs::updateVps($param,$hostinfo);
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
        $snapshotHostModel = new SnapshotVps();
        $list = $snapshotHostModel->where(['host_id'=>$host_id])->select();//
        $data = [];
        foreach ($list as $k=>$v){
            $data[] = ['id'=>$v['id'],'virtuals_id'=>$v['host_id'],'name'=>$v['name'],'created_at'=>$v['create_time']];
        }
        $this->success_qz('succ',$data);
    }

    public function backups_add($host_id){
        $post = $this->request->param();
        $data = Ecs::createBackupHost(['hostid'=>$host_id]);
        if($data['code']==200){
            $this->success_qz('succ',$data);
        }else{
            $this->error_qz($data['msg']);
        }
    }

    public function backups_del($host_id,$id){
        $backupVpsModel = new BackupVps();
        $backupVps = $backupVpsModel->find($id);
        $data = Ecs::removeBackupHost(['hostid'=>$host_id,'id'=>$id]);
        if($data['code']==200){
            $this->success_qz('succ',$data);
        }else{
            $this->error_qz($data['msg']);
        }
    }

    public function backups_restore($host_id,$id){
        $backupVpsModel = new BackupVps();
        $backupVps = $backupVpsModel->find($id);
        $data = Ecs::restoreBackupHost(['hostid'=>$host_id,'id'=>$id]);
        if($data['code']==200){
            $this->success_qz('succ',$data);
        }else{
            $this->error_qz($data['msg']);
        }
    }

    public function backups_list($host_id){
        $backupVpsModel = new BackupVps();
        $list = $backupVpsModel->where(['host_id'=>$host_id])->select();//
        $data = [];
        foreach ($list as $k=>$v){
            $data[] = ['id'=>$v['id'],'virtuals_id'=>$v['host_id'],'name'=>$v['name'],'created_at'=>$v['create_time']];
        }
        $this->success_qz('succ',$data);
    }

    public function mirror_image(){
        $image_model = new ServersImageConfig();
        $line_id = (int)$this->request->param('line_id', 0);
        $list = [];
        if ($line_id > 0) {
            try {
                $list = $image_model->where(['line_id' => $line_id])->select();
            } catch (\Throwable $e) {
                $list = [];
            }
        }
        if (empty($list) || (is_object($list) && method_exists($list, 'isEmpty') && $list->isEmpty())) {
            $list = $image_model->select();
        }
        $data = [];
        foreach ($list as $k=>$v){
            $data[] = [
                'id'=>$v['id'],
                'name'=>$v['os_name'],
                'type'=>$v['os_type_name'],
                'desc'=>$v['remark'],
            ];
        }
        $this->success_qz('succ',$data);
    }

    public function nat_acl_list($host_id){
        $forwardPortModel= new ForwardPortVps();
        $list = $forwardPortModel->where(['host_id'=>$host_id])->order('id desc')->select();

        $this->success_qz('succ',$list);
    }

    public function cdrom($host_id=0){

        $this->success_qz('succ',[]);
    }

    public function security_acl_list($host_id){
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
        $logic = new \app\common\service\Ecs();
        $data = $logic->monitorHost(['hostid'=>$host_id]);
        if($data['code']==200){
            $this->success_qz('success',$data['data']);
        }else{
            $this->error_qz($data['msg']);
        }
    }

    public function reset_os($host_id,$template_id,$password){
        try{
            $hostModel = new HostVps();
            $hostinfo = $hostModel->where(['id'=>$host_id])->find();
            if(!$hostinfo){
                $this->error_qz('云主机不存在');
            }
            $image = new ServersImageConfig();
            $where = "id='".$template_id."' or os_name='".$template_id."'";
            $image_config  = $image->where($where)->find();
            if(empty($image_config)){
                $this->error_qz('镜像不存在');
            }
            $param['os_id'] =$image_config['id'];
            $param['hostid'] = $host_id;
            $param['password'] = ($password=='')?$hostinfo['os_password']:$password;
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
        $data = Ecs::createSnapshotHost(['hostid'=>$host_id]);
        if($data['code']==200){
            $this->success_qz('succ',$data);
        }else{
            $this->error_qz($data['msg']);
        }
    }

    public function snapshot_del($host_id,$id){
        $snapshotHostModel = new SnapshotVps();
        $snapshot = $snapshotHostModel->find($id);
        $data = Ecs::removeSnapshotHost(['hostid'=>$host_id,'id'=>$id]);
        if($data['code']==200){
            $this->success_qz('succ',$data);
        }else{
            $this->error_qz($data['msg']);
        }
    }

    public function snapshot_restore($host_id,$id){
        $snapshotHostModel = new SnapshotVps();
        $snapshot = $snapshotHostModel->find($id);
        $data = Ecs::restoreSnapshotHost(['hostid'=>$host_id,'id'=>$id]);
        if($data['code']==200){
            $this->success_qz('succ',$data);
        }else{
            $this->error_qz($data['msg']);
        }
    }

    public function security_acl_add(){
        $post = $this->request->param();
        $data = Ecs::add_firewall_host($post);
        if($data['code']==200){
            $this->success_qz('succ',$data);
        }else{
            $this->error_qz($data['msg']);
        }
    }

    public function security_acl_del($host_id,$id){
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
        $data = \app\common\service\Ecs::find_port(['keywords'=>$keywords,'hostid'=>$hostid]);
        //查找端口
        return json(['code'=>0,'content'=>$data,'type'=>'success']);
    }

    //添加一个端口
public function add_port_host(){
        $logic = new \app\common\service\Ecs();
        $param = $this->request->param();
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
        $data = $logic->remove_forward_port($param);
        if($data['code']==200){
            $this->success_qz('删除端口成功','');
        }else{
            $this->error_qz($data['msg']);
        }
    }

    public function reset_password($host_id){
        $password = $this->request->post('password');
        $result = Ecs::update_password_host(['hostid'=>$host_id,'password'=>$password]);
        if($result['code']==200){
            $this->success_qz('操作成功','');

        }else{
            $this->error_qz($result['msg']);
        }

    }

    public function vnc_view($host_id){
        $data = Ecs::guidHost(['hostid'=>$host_id]);
        if($data['code']==0){
            $this->json($data['msg'],1);
        }
        if(strstr($data['data']['url'],'vnc')!=false){
            $url = $data['data']['url'].'&host='.$data['data']['host'].'&host_name='.$data['data']['host_name'].'&password='.$data['data']['vnc_password'];
        }else{
            $url = $data['data']['url'];
        }
        header('Location: '.$url);
        exit;
    }

    //自由选配升级
public function elastic_update(){
    try {
        $post = $this->request->param();

        // 必须传 host_id
        if (empty($post['host_id'])) {
            $this->error_qz('缺少 host_id');
        }

        $hostModel = new HostVps();
        $hostinfo  = $hostModel->where(['id' => $post['host_id']])->find();
        if (!$hostinfo) {
            $this->error_qz('云主机不存在');
        }

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

        Ecs::updateVps($param, $hostinfo);
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
        $port = '';
        $remote_ip = $hostinfo->ip;
        //主机ip信息
        $network =[];//ServersIpv4Private
        if($hostinfo->is_nat==1){
            $image_model = new ServersImageConfig();
            $image_info = $image_model->where(['os_name'=>$hostinfo->os_name])->find();
            $port_vps_model = new ForwardPortVps();
            $portlist = $port_vps_model->where(['host_id'=>$hostinfo['id'],'sys'=>2])->column('sport','dport');
            if($image_info->os_type==1){
                $port = $portlist['3389'];
            }else{
                $port =$portlist['22'];
            }
            $node_model = new ServersNode();
            $node = $node_model ->where(['id'=>$hostinfo->node_id])->find();
            $remote_ip = $node->forward_url.':'.$port;
            $natip_model = new ServersIpv4Nat();
            $natip = $natip_model->where(['ip'=>$hostinfo->ip])->find();
            $natip ->public_ip = $node->forward_url;
            //$network
            $network['eth1'] = [$natip];
        }else{
            $ip_model = new ServersIpv4();
            $ip = $ip_model->where(['v_name'=>$hostinfo->host_name])->select();
            $ip = $ip->toArray();
            foreach ($ip as $k=>$v){
                if($v['ip']==$hostinfo['ip']){
                    unset($ip[$k]);
                    array_unshift($ip,$v);
                }
            }
            $network['eth1'] = $ip;
        }



        $privateip_model = new ServersIpv4Private();
        $privateip = $privateip_model->where(['v_name'=>$hostinfo->host_name])->select();
        $network['eth2'] = $privateip;


        $hostinfo->network = $network;
        $hostinfo->remote_ip=$remote_ip;
        $this->success_qz('succ',$hostinfo);
    }
}

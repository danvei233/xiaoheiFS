<?php
namespace certification\mangzhuyun;

use app\admin\lib\Plugin;
use certification\mangzhuyun\logic\Mangzhuyun;
use cmf\phpqrcode\QRcode;

class MangzhuyunPlugin extends Plugin
{
    # 基础信息
    public $info = array(
        'name'        => 'Mangzhuyun',//Demo插件英文名，改成你的插件英文就行了
        'title'       => '芒竹云面容ID验证',
        'description' => '芒竹云手机ID验证二要素/三要素,微信/百度人脸实名认证',
        'status'      => 1,
        'author'      => 'Mangzhuyun',
        'version'     => '2.0',
        'help_url'    => 'https://e.mangzhuyun.cn'
    );

    # 插件安装
    public function install()
    {
        return true;//安装成功返回true，失败false
    }

    # 插件卸载
    public function uninstall()
    {
        return true;//卸载成功返回true，失败false
    }
    /*
         * 数据返回格式
         *  $res = [
         *         'status' => 1, //状态 1已通过，0未通过
         *         'msg' => '', //认证信息
         *         'certify_id' => '认证证书', //可选
         *         'data' => '',  //返回的url链接,没有则为空
         *         'ping' => true,  //是否需要ping轮询 可选 true|false
         *   ]
         *
         */
    # 个人认证
    public function personal($certifi)
    {
        
        $config = $this->getConfig();
        if ($config['type']==2) {
        
        $api=file_get_contents("https://e.mangzhuyun.cn/index/sm_api?key=".$config['key']."&name=".$certifi['name']."&idcard=".$certifi['card']);
        $api=json_decode($api,true);
        if ($api['code'] == 200){
            $requestid=$api['requestid'];
            $data['certify_id']=$requestid;
            if ($api['data']['result']==1) {
            $data['status'] = 1;
            }else {
            $data['status'] = 2;
            $data['auth_fail'] = $api['data']['message']?:'';
            }
            updatePersonalCertifiStatus($data);
        
        }else{
            $data['auth_fail'] = $api['msg']?:'实名认证接口配置错误,请联系管理员';
            return "<h3 class=\"pt-2 font-weight-bold h2 py-4\"><img src=\"\" alt=\"\">{$data['auth_fail']}</h3>";
        }
        }
        if ($config['type']==3) {
        $api=file_get_contents("https://e.mangzhuyun.cn/index/sm3_api?name=".$certifi['name']."&idcard=".$certifi['card']."&mobile=".$certifi['phone']."&key=".$config['key']);
        $api=json_decode($api,true);
        if ($api['code']!=-1) {
        if ($api['code']==200) {
        $data['status'] = 1;
        }else {
        $data['status'] = 2;
        $data['auth_fail']=$api['msg'];
        }
        updatePersonalCertifiStatus($data);
        } else {
        $data['auth_fail'] = $api['msg']?:'实名认证接口配置错误,请联系管理员';
        return "<h3 class=\"pt-2 font-weight-bold h2 py-4\"><img src=\"\" alt=\"\">{$data['auth_fail']}</h3>";
        }
        
        }
        if ($config['type']==2||$config['type']==3) {
        return "<h3 class=\"pt-2 font-weight-bold h2 py-4\"><img src=\"\" alt=\"\"> 正在认证,请稍等...</h3><hr>请勿刷新或退出此页面！";
        }
        if ($config['type']==4) {
        $api=json_decode(file_get_contents("https://e.mangzhuyun.cn/index/sm_wx?url=http://".request()->host()."&key=".$config['key']."&name=".$certifi['name']."&idcard=".$certifi['card']),true);
        //$data = ["status" => 4, "auth_fail" => "", "certify_id" => ""];
        if ($api['code']==200) {
        
            $url= $api["url"];
            $certify_id = $api["token"];
            $data["certify_id"] = $certify_id;
            updatePersonalCertifiStatus($data);
            $uid = \request()->uid;
            $filename = md5($uid . '_zjmf_' . time()) . '.png';
            $file = WEB_ROOT . "upload/{$filename}"; # 临时存放二维码图片
            QRcode::png($url,$file);
            $base64 = base64EncodeImage($file);
            unlink($file);# 删除临时文件
            updatePersonalCertifiStatus($data);
            return "<h5 class=\"pt-2 font-weight-bold h5 py-4\">请使用微信APP扫描二维码</h5><img height='200' width='200' src=\"" . $base64 . "\" alt=\"\">";
        }else {
        $data["auth_fail"] = $api['msg'] ?: "实名认证接口配置错误,请联系管理员";
        return "<h3 class=\"pt-2 font-weight-bold h2 py-4\"><img src=\"\" alt=\"\">" . $data["auth_fail"] . "</h3>";
        }
        //updatePersonalCertifiStatus($data);
        }
        
        if ($config['type']==5) {
        $api=json_decode(file_get_contents("https://e.mangzhuyun.cn/index/bd_sm?key=".$config['key']."&url=http://".request()->host()."&name=".$certifi['name']."&idcard=".$certifi['card']),true);
        //$data = ["status" => 4, "auth_fail" => "", "certify_id" => ""];
        if ($api['code']==200) {
        
            $url= $api["url"];
            $certify_id = $api["token"];
            $data["certify_id"] = $certify_id;
            updatePersonalCertifiStatus($data);
            $uid = \request()->uid;
            $filename = md5($uid . '_zjmf_' . time()) . '.png';
            $file = WEB_ROOT . "upload/{$filename}"; # 临时存放二维码图片
            QRcode::png($url,$file);
            $base64 = base64EncodeImage($file);
            unlink($file);# 删除临时文件
            updatePersonalCertifiStatus($data);
            return "<h5 class=\"pt-2 font-weight-bold h5 py-4\">请使用微信/QQ/百度APP/芒竹EID验证程序APP等应用扫描二维码</h5><img height='200' width='200' src=\"" . $base64 . "\" alt=\"\">";
        }else {
        $data["auth_fail"] = $api['msg'] ?: "实名认证接口配置错误,请联系管理员";
        return "<h3 class=\"pt-2 font-weight-bold h2 py-4\"><img src=\"\" alt=\"\">" . $data["auth_fail"] . "</h3>";
        }
        
        }
    }

    # 前台自定义字段输出
    public function collectionInfo()
    {
        $config = $this->getConfig();/*
		$data = [
            'cert_type' => [
                'title' => '证件类型',
                'type'  => 'select',
                'options' => [
					'IDENTITY_CARD'=>'中国居民身份证',
					'HOME_VISIT_PERMIT_HK_MC'=>'港澳通行证',
					'HOME_VISIT_PERMIT_TAIWAN'=>'台湾通行证',
					'RESIDENCE_PERMIT_HK_MC'=>'港澳居住证',
					'RESIDENCE_PERMIT_TAIWAN'=>'台湾居住证',
				],
                'tip'   => '',
                'required'   => true, # 是否必填
            ],
            ];*/
            if ($config['type']==3) {
            $data=[
            'phone' => [
                    'title' => '手机号',
                    'type'  => 'text',
                    'value' => '',
                    'tip'   => '请输入手机号',
                    'required'   => true, # 是否必填
                ],
            
        ];
            }
            
		/* $data['cert_type']['cert_type']=[
			'IDENTITY_CARD'=>'身份证',
			'HOME_VISIT_PERMIT_HK_MC'=>'港澳通行证',
			'HOME_VISIT_PERMIT_TAIWAN'=>'台湾通行证',
			'RESIDENCE_PERMIT_HK_MC'=>'港澳居住证',
			'RESIDENCE_PERMIT_TAIWAN'=>'台湾居住证',
		];  */
        return $data;
    }

    # 当返回数据中ping为true时,需要实现此方法,系统轮询调用
    public function getStatus($certifi)
    {
        $config = $this->getConfig();
        if ($config['type']==4) {
       
        $ojw=$certifi["certify_id"];
        $api1=file_get_contents("https://e.mangzhuyun.cn/index/wx_cx?key=".$config['key']."&token=".$ojw);
        $api=json_decode($api1,true);
        if ($api['sm']==0) {
        $sm=4;
        }
        if ($api['sm']==3) {
        $sm=2;
        }
        
        if ($api['sm']==1) {
        $sm=1;
        }
        $res=[
            'status'=>$sm,
            'msg'=>$api['msg']
            ];
        return $res;
        }
        if ($config['type']==5) {
       
        $ojw=$certifi["certify_id"];
        $api1=file_get_contents("https://e.mangzhuyun.cn/index/bd_cx?key=".$config['key']."&token=".$ojw);
        $api=json_decode($api1,true);
        if ($api['sm']==0) {
        $sm=4;
        }
        if ($api['sm']==1) {
        $sm=2;
        }
        
        if ($api['sm']==2) {
        $sm=1;
        }
        $res=[
            'status'=>$sm,
            'msg'=>$api['msg']
            ];
        return $res;
        }
    
    }
}
<?php
/*
 * 实名认证后台自定义配置
 */
return [
    # 系统默认字段
    'module_name'=>[
        'title'=>'名称',
        'type' =>'text',
        'value'=>'芒竹云实名认证插件',
        'tip'  =>'前台显示的名称'
        ],
    'amount' => [ # 无此配置,默认为0
        'title' => '金额',
        'type'  => 'text',
        'value' => 0,
        'tip'   => '支付金额',
    ],
    'free' => [ # 无此配置,默认为0
        'title' => '免费认证次数',
        'type'  => 'text',
        'value' => 0,
        'tip'   => '免费认证次数',
    ],
    # 开发者自定义字段
    'type'        => [// 在后台插件配置表单中的键名 ,会是config[select]
        'title'   => '认证方式',
        'type'    => 'select',
        'options' => [//select 和radio,checkbox的子选项
            2 => '两要素', // 值=>显示  姓名 银行卡号
            3 => '三要素', //  姓名 身份证号 银行卡号
            4 => 'WeChat面容ID', 
            5 => 'Baidu面容ID',
        ],
        'value'   => '5',
        'tip'     => '认证方式',
    ],
    'key'      => [
        'title' => 'KEY密钥',
        'type'  => 'text',
        'value' => '',
        'tip'   => '',
    ],
    
];

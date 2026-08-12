#!/usr/bin/env php
<?php
/**
 * 生成 cloud_credentials 配置项所需的凭据散列。
 *
 * 用法：
 *   php qz-override/scripts/make-credential-hash.php <credential-id> <api-key>
 *
 * 输出 `hash` 值：hash_hmac('sha256', <api-key>, <credential-id>)。
 * 明文 api-key 只在此处出现，配置里只存散列（revoked/scope/user_id/quota 见
 * qz-override/tests/CloudPolicyTest.php 中的示例结构）。
 */

if ($argc !== 3) {
    fwrite(STDERR, "usage: php make-credential-hash.php <credential-id> <api-key>\n");
    exit(1);
}
$id = $argv[1];
$key = $argv[2];
if ($id === '' || $key === '') {
    fwrite(STDERR, "credential-id and api-key must not be empty\n");
    exit(1);
}
echo hash_hmac('sha256', $key, $id), "\n";

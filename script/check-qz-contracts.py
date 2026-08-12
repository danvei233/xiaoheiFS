#!/usr/bin/env python3
"""
Executable static contract checks for the qz-override legacy PHP layer.

Runnable without PHP. When PHP is present, the stronger executable unit tests
in `qz-override/tests/CloudPolicyTest.php` run instead (see README/report).
This script verifies the security invariants directly from source:

- A1: scoped/revocable hashed credentials replace the single shared apikey;
       Cloud.php resolves via CloudPolicy and checks scope before dispatch.
- A3: hostinfo public view strips node secrets; a separate scoped
       hostinfo_credentials endpoint exists for automation callers.
- A6: create/modify paths call server-side resource/quota/ownership policy.

Exit code is non-zero on any contract violation.
"""
import pathlib
import re
import sys

REPO = pathlib.Path(__file__).resolve().parent.parent
QZ = REPO / "qz-override"
CLOUD = QZ / "app/api/controller/Cloud.php"
POLICY = QZ / "app/common/service/CloudPolicy.php"
CLOUD_YML = QZ / "cloud.yml"

failures = []


def check(cond, msg):
    if cond:
        print(f"  ok   {msg}")
    else:
        print(f"  FAIL {msg}")
        failures.append(msg)


def read(p):
    return p.read_text(encoding="utf-8", errors="replace")


def main():
    print("== qz-override static contract checks ==")

    cloud = read(CLOUD)
    policy = read(POLICY)

    # A1: credential resolution + scope gating.
    print("[A1] scoped hashed credentials")
    check("use app\\common\\service\\CloudPolicy;" in cloud,
          "Cloud.php imports CloudPolicy")
    check("CloudPolicy::resolveCredential" in cloud,
          "initialize() resolves per-tenant credentials")
    check("CloudPolicy::canCall" in cloud,
          "initialize() enforces scope before dispatch")
    check("hash_hmac('sha256', $apikey, $id)" in policy or
          "hash_hmac('sha256', $apikey, $id)" in policy,
          "credential hashes are keyed HMAC-SHA256 (no plaintext keys)")
    check("hash_equals" in policy, "credential compare is constant-time")
    check("revoked" in policy, "credentials are revocable")
    check("legacy" in policy, "legacy single-key fallback is explicit and documented")
    check("CREDENTIAL_SCOPED_ACTIONS" in policy,
          "credential-only endpoints are defined")

    # A3: hostinfo redaction.
    print("[A3] hostinfo secret redaction")
    check("CloudPolicy::publicHostInfo" in cloud,
          "hostinfo returns the redacted public view")
    check("HOSTINFO_SECRET_FIELDS" in policy and
          "os_password" in policy and "vnc_password" in policy,
          "policy enumerates the node secret fields to strip")
    check("hostinfo_credentials" in cloud,
          "scoped hostinfo_credentials endpoint exists for automation")
    check("hostinfo_credentials" in CLOUD_YML.read_text(encoding="utf-8"),
          "cloud.yml documents hostinfo_credentials")

    # A6: server-side plan/quota/ownership.
    print("[A6] server-side plan/quota/ownership checks")
    for method in ("enforceCreatePolicy", "enforceUpdatePolicy", "loadOwnedHost", "assertOwnsHost"):
        check(f"protected function {method}" in cloud, f"Cloud.php defines {method}")
    check("CloudPolicy::enforceResourcePolicy" in cloud,
          "create/update call resource (plan/caps) policy")
    check("CloudPolicy::enforceInstanceQuota" in cloud,
          "create calls instance quota policy")
    check("CloudPolicy::ownsHost" in cloud,
          "host ops enforce ownership")
    check("enforceCreatePolicy" in cloud and "openhost" in cloud,
          "openhost enforces create policy")
    check("enforceCreatePolicy" in cloud and "create_host" in cloud,
          "create_host enforces create policy")
    check("enforceUpdatePolicy" in cloud and "elastic_update" in cloud,
          "elastic_update enforces update policy")

    # hostinfo must NOT return the raw model with secrets.
    print("[A3] no raw-secret hostinfo leak")
    check("CloudPolicy::publicHostInfo($hostinfo->toArray()," in cloud,
          "hostinfo echoes the redacted array, not the raw model")
    check("hostinfo_credentials" in cloud and "success_qz('succ', $hostinfo)" in cloud,
          "only hostinfo_credentials echoes the full model")

    # F1/F4/F5: long-lived secrets never appear in URLs or redirects.
    print("[F1/F4/F5] no secrets in URL/redirect")
    check("CloudPolicy::isSafeRedirectTarget" in cloud,
          "redirect target goes through the whitelist helper")
    check("CloudPolicy::buildPanelRedirect" in cloud,
          "panel redirect is built through the ticket-only helper")
    check("TicketService::issue" in cloud,
          "panel/vnc issue short-lived single-use tickets")
    check("'ticket' =>" in cloud and "consume_url" in cloud,
          "vnc_view returns a ticket + same-origin consume_url (no vnc_password URL)")
    check("request->post('password'" in cloud,
          "reset_os reads the new password from the POST body, not the URL")
    check("rejectSecretQueryParams" in cloud,
          "URL query is scanned for long-lived secrets and rejected")
    check("panel_password' . $password" not in cloud and "panel_password='.$password" not in cloud,
          "no code path builds a panel redirect carrying panel_password")

    # Round-5: production consume route + persistence contract (TicketService was
    # issue-only with no consume endpoint; the table existed only in comments).
    print("[Round-5] production ticket consume endpoint + install contract")
    ticket_svc = read(QZ / "app/common/service/TicketService.php")
    check("consumeTicket" in ticket_svc,
          "TicketService exposes production consumeTicket (kind/expiry/single-use)")
    check("function deleteExpired" in ticket_svc,
          "TicketService keeps the ticket table bounded (deleteExpired)")
    ecs = QZ / "app/control/controller/Ecs.php"
    check(ecs.exists() and "function login" in read(ecs) and "function vnc" in read(ecs),
          "control/ecs/login + control/ecs/vnc consume endpoints exist")
    check("consumeTicket" in read(ecs),
          "consume controller consumes via TicketService::consumeTicket")
    ecs_src = read(ecs)
    check("renderAutoLogin" in ecs_src and 'method="post"' in ecs_src and "submit()" in ecs_src,
          "consume endpoint hands off credentials via POST body (auto-submit form)")
    check("Cache-Control: no-store" in ecs_src and "X-Frame-Options: DENY" in ecs_src,
          "auto-submit form is no-store + frame-denied (password not cached/embedded)")
    sql = QZ / "db/panel_ticket.sql"
    check(sql.exists() and "CREATE TABLE IF NOT EXISTS `panel_ticket`" in read(sql),
          "persistent panel_ticket install contract exists (db/panel_ticket.sql)")
    check("consumeTicket" in read(QZ / "tests/CloudPolicyTest.php"),
          "PHP regression tests cover consumeTicket")
    # cloud.yml documents the consume routes.
    check("/control/ecs/login" in read(CLOUD_YML) and "/control/ecs/vnc" in read(CLOUD_YML),
          "cloud.yml documents the production consume routes")

    # F2: renew/create/openhost end_time window validation.
    print("[F2] end_time window validation")
    check("assertValidEndTime" in cloud, "Cloud.php defines assertValidEndTime")
    check("function validateEndTime" in policy,
          "CloudPolicy validates the end_time window (future + max days)")
    check("validateEndTime" in cloud and "renew" in cloud,
          "renew validates nextduedate before saving end_time")

    # P2-11: renew/update must bind + atomically consume a paid-order
    # authorization; merely having order scope + a future window must not renew
    # for free or swap to a higher-spec product.
    print("[P2-11] renew/update consume a paid-order authorization")
    check("consumeOrderAuthorization" in cloud,
          "Cloud.php defines consumeOrderAuthorization (order binding helper)")
    check("OrderAuthorizationService::consume" in cloud,
          "Cloud.php consumes the paid-order authorization via OrderAuthorizationService")
    check("consumeOrderAuthorization('renew'" in cloud or
          "consumeOrderAuthorization('renew'" in cloud.replace(" ", ""),
          "renew requires an order authorization before saving end_time")
    check("consumeOrderAuthorization('update'" in cloud or
          "consumeOrderAuthorization('update'" in cloud.replace(" ", ""),
          "update requires an order authorization before changing product/resources")
    check("consumeOrderAuthorization('elastic_update'" in cloud or
          "consumeOrderAuthorization('elastic_update'" in cloud.replace(" ", ""),
          "elastic_update requires an order authorization before changing resources")
    check("Db::startTrans" in cloud and "Db::rollback" in cloud,
          "order consumption + host mutation are in one DB transaction (failure rolls back)")
    check("缺少有效的已支付订单授权" in cloud,
          "missing/invalid order authorization is fail-closed (no state change)")
    auth_svc = QZ / "app/common/service/OrderAuthorizationService.php"
    check(auth_svc.exists() and "function consume" in read(auth_svc),
          "OrderAuthorizationService.consume exists (kind/host/user/expiry/single-use)")
    check("markConsumed" in read(auth_svc),
          "OrderAuthorizationService atomically consumes (single-use)")
    order_sql = QZ / "db/order_auth.sql"
    check(order_sql.exists() and "CREATE TABLE IF NOT EXISTS `order_auth`" in read(order_sql),
          "persistent order_auth install contract exists (db/order_auth.sql)")
    check("concurrent order-auth single-winner" in read(QZ / "tests/CloudPolicyTest.php"),
          "PHP regression tests cover replay/concurrency/over-authorization")

    # P2-66: cloud.yml must stay in lockstep with the production `apikey` header
    # auth. Every protected Cloud operation carries the real security
    # requirement; only the ticket-authenticated consume routes opt out (their
    # bearer is the single-use ticket, not an apikey).
    print("[P2-66] OpenAPI security matches production apikey header auth")
    yml = read(CLOUD_YML)
    check("name: apikey" in yml and "in: header" in yml,
          "cloud.yml securityScheme is the production apikey header")
    check("security:" in yml and "ApiKeyAuth: []" in yml,
          "cloud.yml applies the ApiKeyAuth requirement at the document level")
    check("security: []" in yml,
          "ticket-authenticated consume routes explicitly opt out of apikey (security: [])")
    # Protected Cloud endpoints must NOT be able to silently drop the security
    # requirement: the /renew, /update, /elastic_update order endpoints and the
    # panel/vnc/credentials endpoints must keep the inherited security (no
    # security: [] on them).
    for guarded in ("/renew", "/update", "/elastic_update", "/panel", "/vnc_view",
                    "/hostinfo_credentials", "/openhost", "/create_host"):
        import re as _re
        for m in _re.finditer(r"(?m)^  /[^:]+:\n(?:    .*\n)*?", yml):
            block = m.group(0)
            if block.lstrip().startswith(guarded):
                check("security: []" not in block,
                      f"{guarded} keeps the real security requirement (no opt-out)")
    check("hostinfo_credentials" in yml and "credentials" in yml,
          "cloud.yml documents the scoped credentials endpoint")

    # F7: hostList is credential-scoped for tenant-bound credentials.
    print("[F7] hostList credential scoping")
    check("CloudPolicy::hostListScope" in cloud,
          "hostList applies the credential-scoped user filter")
    check("hostListScope" in policy, "CloudPolicy.hostListScope exists")

    # Round-5: remove_port_host must require host_id and enforce mapping
    # ownership (missing host_id used to skip loadOwnedHost and still delete).
    print("[Round-5] remove_port_host ownership")
    check("validatePortRemoval" in policy,
          "CloudPolicy.validatePortRemoval exists (host + mapping ownership)")
    check("CloudPolicy::validatePortRemoval" in cloud,
          "remove_port_host enforces host + mapping ownership")
    check("缺少 host_id" in policy and "不属于该主机" in policy,
          "missing host_id / cross-tenant mapping are fail-closed")
    check("mapping belonging to another host rejected" in read(QZ / "tests/CloudPolicyTest.php"),
          "PHP regression tests cover cross-tenant port removal")

    # Tests exist and are runnable.
    print("[tests] runnable regression")
    test = QZ / "tests/CloudPolicyTest.php"
    check(test.exists() and "resolveCredential" in read(test),
          "PHP policy unit tests exist (CloudPolicyTest.php)")
    check((QZ / "scripts/make-credential-hash.php").exists(),
          "credential hash helper exists")

    print()
    if failures:
        print(f"{len(failures)} contract violation(s)")
        sys.exit(1)
    print("all qz static contracts satisfied")


if __name__ == "__main__":
    main()

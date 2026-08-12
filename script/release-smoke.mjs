import fs from 'node:fs'
import path from 'node:path'
import { fileURLToPath } from 'node:url'

const scriptDir = path.dirname(fileURLToPath(import.meta.url))
const repoRoot = path.resolve(scriptDir, '..')
const backendRoot = path.join(repoRoot, 'backend')
const manifestRoot = path.join(backendRoot, 'plugins')
const mappingPath = path.join(scriptDir, 'plugin-sources.txt')

function readRepoFile(name) {
  return fs.readFileSync(path.join(repoRoot, name), 'utf8')
}

function repoFileOr(name, fallback) {
  return fs.existsSync(path.join(repoRoot, name))
    ? fs.readFileSync(path.join(repoRoot, name), 'utf8')
    : fallback
}

function expectText(text, expected, source) {
  if (!text.includes(expected)) throw new Error(`${source}: missing ${expected}`)
}

function rejectText(text, rejected, source) {
  if (text.includes(rejected)) throw new Error(`${source}: contains forbidden ${rejected}`)
}

const mappings = new Map()
// 已知 test-only 插件（历史硬编码），与 manifest 的 test_only 字段共同兜底。
const excludedManifests = new Set(['payment/mockpay'])
const testOnlyManifests = new Set()
for (const rawLine of fs.readFileSync(mappingPath, 'utf8').split(/\r?\n/)) {
  const line = rawLine.trim()
  if (!line || line.startsWith('#')) continue
  const [pluginId, packagePath, extra] = line.split('|')
  if (!pluginId || !packagePath || extra !== undefined) {
    throw new Error(`invalid plugin mapping: ${rawLine}`)
  }
  if (mappings.has(pluginId)) throw new Error(`duplicate plugin mapping: ${pluginId}`)
  mappings.set(pluginId, packagePath)
}

const manifests = new Map()
for (const category of fs.readdirSync(manifestRoot, { withFileTypes: true })) {
  if (!category.isDirectory()) continue
  const categoryDir = path.join(manifestRoot, category.name)
  for (const plugin of fs.readdirSync(categoryDir, { withFileTypes: true })) {
    if (!plugin.isDirectory()) continue
    const pluginId = `${category.name}/${plugin.name}`
    const manifestPath = path.join(categoryDir, plugin.name, 'manifest.json')
    if (!fs.existsSync(manifestPath)) continue

    const manifest = JSON.parse(fs.readFileSync(manifestPath, 'utf8'))
    if (manifest.plugin_id !== plugin.name) {
      throw new Error(`${pluginId}: manifest plugin_id is ${manifest.plugin_id}`)
    }
    if (manifest.test_only === true) {
      testOnlyManifests.add(pluginId)
    }
    for (const [platform, binaryPath] of Object.entries(manifest.binaries ?? {})) {
      const suffix = platform.startsWith('windows_') ? 'plugin.exe' : 'plugin'
      if (binaryPath !== `bin/${platform}/${suffix}`) {
        throw new Error(`${pluginId}: invalid ${platform} binary path ${binaryPath}`)
      }
    }
    manifests.set(pluginId, manifestPath)
  }
}

for (const [pluginId, packagePath] of mappings) {
  if (excludedManifests.has(pluginId)) throw new Error(`${pluginId}: test-only plugin must not be released`)
  if (testOnlyManifests.has(pluginId)) throw new Error(`${pluginId}: manifest marked test_only must not be in the release mapping`)
  if (!manifests.has(pluginId)) throw new Error(`${pluginId}: mapping has no manifest`)
  const sourceDir = path.resolve(backendRoot, packagePath)
  if (!fs.statSync(sourceDir).isDirectory()) throw new Error(`${pluginId}: source is not a directory`)
}
for (const pluginId of manifests.keys()) {
  if (excludedManifests.has(pluginId) || testOnlyManifests.has(pluginId)) continue
  if (!mappings.has(pluginId)) throw new Error(`${pluginId}: manifest has no source mapping`)
}
// 已知 test-only 插件必须显式声明 test_only，防止新引入的 mock 插件被误发布。
for (const pluginId of excludedManifests) {
  if (!testOnlyManifests.has(pluginId)) {
    throw new Error(`${pluginId}: known test-only plugin must carry "test_only": true in its manifest`)
  }
}

const releaseWorkflow = readRepoFile('.github/workflows/release-build.yml')
expectText(releaseWorkflow, 'ref: ${{ needs.prepare.outputs.tag }}', 'release-build.yml')
expectText(releaseWorkflow, 'git show-ref --verify --quiet "refs/tags/$RELEASE_TAG"', 'release-build.yml')
expectText(releaseWorkflow, './script/release-smoke.sh', 'release-build.yml')
expectText(releaseWorkflow, './script/build-plugins.sh windows amd64', 'release-build.yml')
expectText(releaseWorkflow, './script/build-plugins.sh "$TARGET_OS" amd64', 'release-build.yml')
expectText(releaseWorkflow, 'mkdir -p "$output_dir/static" "$output_dir/plugins"', 'release-build.yml')
rejectText(releaseWorkflow, 'GOOS=linu\n', 'release-build.yml')
rejectText(releaseWorkflow, 'go env -w', 'release-build.yml')
rejectText(releaseWorkflow, 'cp -a backend/plugins', 'release-build.yml')

for (const dockerfile of ['docker/Dockerfile', 'docker/Dockerfile.alpine']) {
  const dockerfileText = readRepoFile(dockerfile)
  expectText(dockerfileText, '/app/script/build-plugins.sh', dockerfile)
  expectText(dockerfileText, '/app/runtime-plugins ./bundled-plugins', dockerfile)
  rejectText(dockerfileText, 'backend/plugin ./plugin', dockerfile)
}

const compose = readRepoFile('docker/docker-compose.yml')
for (const mount of [
  'app-config:/app/config',
  'app-uploads:/app/uploads',
  'app-plugins:/app/plugins',
  'app-data:/app/data',
  'mysql-data:/var/lib/mysql'
]) {
  expectText(compose, mount, 'docker-compose.yml')
}
expectText(compose, 'restart: unless-stopped', 'docker-compose.yml')
expectText(compose, '/api/v1/install/status', 'docker-compose.yml')
expectText(compose, '/api/v1/site/settings', 'docker-compose.yml')

const pingbotWorkflow = readRepoFile('.github/workflows/release-pingbot-probe.yml')
expectText(pingbotWorkflow, 'cp pingbot/deploy/systemd/install.sh "$package_dir/"', 'release-pingbot-probe.yml')
expectText(pingbotWorkflow, 'needs: [prepare, build]', 'release-pingbot-probe.yml')
expectText(pingbotWorkflow, 'git show-ref --verify --quiet "refs/tags/$RELEASE_TAG"', 'release-pingbot-probe.yml')

// Flutter release workflows must (a) materialize the release keystore on the
// runner (not just check an env var for a path that no longer exists) and
// (b) inject the real update/API domains and the update public key via
// --dart-define so a release never ships the placeholder domains silently.
for (const wf of ['release-xiaoheifs-app.yml', 'release-xiaoheifs-userapp.yml']) {
  const wfText = readRepoFile(`.github/workflows/${wf}`)
  expectText(wfText, 'Provision release keystore', wf)
  expectText(wfText, 'ANDROID_RELEASE_KEYSTORE_B64', wf)
  expectText(wfText, 'base64 -d', wf)
  expectText(wfText, '! -s "$KEYSTORE_PATH"', wf)
  expectText(wfText, 'Resolve build-time dart-defines', wf)
  expectText(wfText, '--dart-define=UPDATE_SERVER_URL', wf)
  expectText(wfText, '--dart-define=UPDATE_SIGN_PUBLIC_KEY', wf)
  expectText(wfText, 'flutter build apk --release $DART_DEFINES', wf)
  // A real `release` event must fail the build when the update URL / public
  // key are not configured, so a release never ships a placeholder-domain or
  // placeholder-key APK with a dead signed-update channel.
  expectText(wfText, "github.event_name == 'release'", wf)
  expectText(wfText, 'UPDATE_SERVER_URL is required for a release build', wf)
  expectText(wfText, 'UPDATE_SIGN_PUBLIC_KEY is required for a release build', wf)
}
// The user app release also injects the default API URL.
const userappWf = readRepoFile('.github/workflows/release-xiaoheifs-userapp.yml')
expectText(userappWf, '--dart-define=DEFAULT_API_URL', 'release-xiaoheifs-userapp.yml')

// The signing contract documents how to host the signed manifest so the
// end-to-end update channel is actually runnable, not a dead feature.
const signingContract = readRepoFile('docs/update-signing-contract.md')
expectText(signingContract, '发布端托管', 'update-signing-contract.md')
expectText(signingContract, '静态托管', 'update-signing-contract.md')

// The Docker CI must actually run the client security regression tests.
const dockerWf = readRepoFile('.github/workflows/docker-build.yml')
expectText(dockerWf, 'test-clients', 'docker-build.yml')
expectText(dockerWf, 'dart test', 'docker-build.yml')
expectText(dockerWf, 'CloudPolicyTest.php', 'docker-build.yml')
expectText(dockerWf, 'check-qz-contracts.py', 'docker-build.yml')
expectText(dockerWf, 'test-sync-plugins.sh', 'docker-build.yml')
expectText(dockerWf, 'GOWORK=off go test ./internal/adapter/plugins/core/...', 'docker-build.yml')
// The real flutter_test suites of both apps must run in CI, not just the
// pure-Dart verifier.
expectText(dockerWf, 'flutter-test', 'docker-build.yml')
expectText(dockerWf, 'subosito/flutter-action', 'docker-build.yml')
expectText(dockerWf, 'flutter test', 'docker-build.yml')

const entrypoint = readRepoFile('docker/entrypoint.sh')
expectText(entrypoint, 'xiaohei-sync-plugins', 'docker/entrypoint.sh')
// The entrypoint must never fall back to an unconditional copy when the sync
// script is present; a raw cp would silently overwrite admin-managed plugins.
rejectText(entrypoint, 'cp -a bundled-plugins/. plugins/', 'docker/entrypoint.sh')
// Plugin sync is fail-closed: a managed plugin that cannot be brought to the
// bundle baseline must not let the container start a stale plugin set.
expectText(entrypoint, 'refusing to start', 'docker/entrypoint.sh')
expectText(entrypoint, 'xiaohei-sync-plugins --verify', 'docker/entrypoint.sh')
expectText(entrypoint, 'exit 1', 'docker/entrypoint.sh')
const syncScript = readRepoFile('docker/sync-plugins.sh')
expectText(syncScript, 'PLUGIN_BUNDLE_STATE', 'docker/sync-plugins.sh')
expectText(syncScript, 'copy_plugin', 'docker/sync-plugins.sh')
expectText(syncScript, 'admin', 'docker/sync-plugins.sh')
expectText(syncScript, '--verify', 'docker/sync-plugins.sh')
// Whole-round failure returns non-zero (no silent warn-and-continue).
expectText(syncScript, 'exit 1', 'docker/sync-plugins.sh')
for (const dockerfile of ['docker/Dockerfile', 'docker/Dockerfile.alpine']) {
  expectText(readRepoFile(dockerfile), 'xiaohei-sync-plugins', dockerfile)
}

const pingbotInstall = readRepoFile('pingbot/deploy/systemd/install.sh')
expectText(pingbotInstall, 'REPLACE_CONFIG', 'pingbot install.sh')
expectText(pingbotInstall, 'keep existing', 'pingbot install.sh')
expectText(pingbotInstall, 'systemctl restart pingbot.service', 'pingbot install.sh')

const eslintConfig = readRepoFile('frontend/eslint.config.mjs')
expectText(eslintConfig, "'.vite-cache/**'", 'eslint.config.mjs')

// Non-root runtime: the images must not leave the web server or its plugin
// children running as root. The entrypoint hands the runtime dirs to the app
// user and drops privileges (su-exec on Alpine, setpriv on Debian) before exec.
const entrypointSh = readRepoFile('docker/entrypoint.sh')
expectText(entrypointSh, 'id -u', 'entrypoint.sh')
expectText(entrypointSh, 'su-exec', 'entrypoint.sh')
expectText(entrypointSh, 'setpriv', 'entrypoint.sh')
expectText(entrypointSh, 'chown', 'entrypoint.sh')

for (const dockerfile of ['docker/Dockerfile', 'docker/Dockerfile.alpine']) {
  const dockerfileText = readRepoFile(dockerfile)
  expectText(dockerfileText, 'xiaohei', dockerfile)
  expectText(dockerfileText, 'chown -R xiaohei:xiaohei /app', dockerfile)
}

// Compose MySQL healthcheck must not embed the password on the mysqladmin
// command line: word-splitting breaks on shell metacharacters and the password
// leaks into the process list. mysqladmin ping only detects server aliveness.
rejectText(compose, '-p$$MYSQL_ROOT_PASSWORD', 'docker-compose.yml')
rejectText(compose, '-p$MYSQL_ROOT_PASSWORD', 'docker-compose.yml')

// The Linux build script must build plugins too, so the local build output is a
// complete release package instead of a bare server without plugins.
const buildLinux = readRepoFile('script/build-linux.sh')
expectText(buildLinux, 'build-plugins.sh', 'build-linux.sh')

// build-plugins.sh must reject any output directory inside backend/plugins
// (or equal to it), not just the exact manifest root.
const buildPlugins = readRepoFile('script/build-plugins.sh')
expectText(buildPlugins, '"$MANIFEST_ROOT"/*', 'build-plugins.sh')
// P2-65: every bundled plugin must be signed (checksums.json + signature.sig)
// so the host can import it on image upgrade; unsigned plugins are skipped
// explicitly, never silently.
expectText(buildPlugins, 'sign_plugin', 'build-plugins.sh')
expectText(buildPlugins, 'pluginsign', 'build-plugins.sh')
expectText(buildPlugins, 'checksums.json', 'build-plugins.sh')
expectText(buildPlugins, 'signature.sig', 'build-plugins.sh')
// P2-65 fail-closed: the build must refuse to produce a bundle when no signing
// key is present (never a generated dev keypair passed off as a release), and
// the key must reach pluginsign through a protected source (env), not argv.
expectText(buildPlugins, 'PLUGIN_SIGN_PRIVATE_KEY', 'build-plugins.sh')
expectText(buildPlugins, 'refusing to build an unsigned/dev-signed', 'build-plugins.sh')
expectText(buildPlugins, 'PLUGIN_SIGN_PRIVATE_KEY="${PLUGIN_SIGN_PRIVATE_KEY:-$PLUGIN_SIGNING_KEY}"', 'build-plugins.sh')
const pluginsignMain = readRepoFile('backend/cmd/tools/pluginsign/main.go')
expectText(pluginsignMain, 'PLUGIN_SIGN_PRIVATE_KEY', 'pluginsign main.go')
expectText(pluginsignMain, 'ed25519-priv-stdin', 'pluginsign main.go')
const bootstrap = readRepoFile('backend/internal/adapter/plugins/core/bootstrap.go')
expectText(bootstrap, 'not official', 'bootstrap.go')
expectText(bootstrap, 'log.Printf', 'bootstrap.go')
// P2-65 closure: a real release/upgrade E2E contract test runs build-plugins.sh,
// round-trips the signature and pins bootstrap import (signed imports, unsigned
// logged-and-skipped) — not just a Docker state hash.
const releaseSigningTest = readRepoFile('script/test-release-plugin-signing.sh')
expectText(releaseSigningTest, 'build-plugins.sh', 'test-release-plugin-signing.sh')
expectText(releaseSigningTest, '-verify -ed25519-pub', 'test-release-plugin-signing.sh')
expectText(releaseSigningTest, 'tampered binary fails verification', 'test-release-plugin-signing.sh')
const pluginsign = readRepoFile('backend/cmd/tools/pluginsign/main.go')
expectText(pluginsign, 'verifyPlugin', 'pluginsign main.go')
expectText(pluginsign, '-verify', 'pluginsign main.go')
expectText(repoFileOr('backend/internal/adapter/plugins/core/bootstrap_upgrade_test.go', ''),
          'TestBootstrapFromDiskImageUpgrade',
          'bootstrap_upgrade_test.go')

// P0-24: the build context must never carry deployment secrets. The dedicated
// check-dockerignore-contract.py simulates Docker's exclusion rules over a
// representative secret set; the static assertion below keeps the .dockerignore
// honest even if the runtime checker is not wired into a pipeline.
const dockerignore = readRepoFile('.dockerignore')
for (const needle of [
  'app.config.yaml', '*.db', '*.sqlite', '*.sqlite3',
  '*.key', '*.pem', '*.jks', '*.p12', '*.pfx', '*.keystore', '*.crt',
  '.env.*',
]) {
  expectText(dockerignore, needle, '.dockerignore')
}
rejectText(dockerignore, '!*', '.dockerignore') // no blanket re-inclusion

// P2-84: CI gates must cover the real backend suite (go test ./... + go vet)
// and the real frontend suite (typecheck/lint/build), not just the clients/core
// smoke packages.
expectText(dockerWf, 'backend-go', 'docker-build.yml')
expectText(dockerWf, 'go test ./...', 'docker-build.yml')
expectText(dockerWf, 'go vet ./...', 'docker-build.yml')
expectText(dockerWf, 'frontend-web', 'docker-build.yml')
expectText(dockerWf, 'npm run typecheck', 'docker-build.yml')
expectText(dockerWf, 'npm run lint', 'docker-build.yml')
expectText(dockerWf, 'npm run build', 'docker-build.yml')
// P2-83: Python SDK transport tests run in CI.
expectText(dockerWf, 'Python SDK transport', 'docker-build.yml')
expectText(dockerWf, 'unittest discover', 'docker-build.yml')

// P2-84 workflow static contract: every third-party action is pinned to a full
// 40-char commit SHA (no floating tags / mutable branches), and the real
// backend + frontend + flutter gates exist as PR jobs (execution is validated by
// GitHub Actions itself — a broken workflow cannot pass a PR).
const allWorkflows = ['docker-build.yml', 'release-build.yml', 'release-pingbot-probe.yml',
                      'release-xiaoheifs-app.yml', 'release-xiaoheifs-userapp.yml']
for (const wfName of allWorkflows) {
  const wf = readRepoFile(`.github/workflows/${wfName}`)
  const actionRefs = [...wf.matchAll(/uses:\s+([^\s@]+)@([^\s]+)/g)]
  for (const m of actionRefs) {
    const [, name, ref] = m
    if (name.startsWith('./')) continue // local action
    if (!/^[0-9a-f]{40}$/.test(ref)) {
      throw new Error(`${wfName}: third-party action ${name}@${ref} is not pinned to a 40-char SHA`)
    }
  }
}
for (const [wfName, gates] of [
  ['docker-build.yml', ['backend-go', 'go test ./...', 'go vet ./...',
                        'frontend-web', 'npm run typecheck', 'npm run lint', 'npm run build',
                        'flutter-test', 'flutter test', 'test-clients',
                        'release plugin-signing E2E contract',
                        'check-dockerignore-contract.py']],
]) {
  const wf = readRepoFile(`.github/workflows/${wfName}`)
  for (const gate of gates) expectText(wf, gate, wfName)
}
// The CI jobs must execute on PRs (draft PRs excepted), not only on main.
expectText(dockerWf, "github.event.pull_request.draft == false", 'docker-build.yml')

console.log(`release smoke: ${mappings.size} plugin mappings cover ${manifests.size} manifests`)

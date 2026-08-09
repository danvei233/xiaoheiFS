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

function expectText(text, expected, source) {
  if (!text.includes(expected)) throw new Error(`${source}: missing ${expected}`)
}

function rejectText(text, rejected, source) {
  if (text.includes(rejected)) throw new Error(`${source}: contains forbidden ${rejected}`)
}

const mappings = new Map()
const excludedManifests = new Set(['payment/mockpay'])
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
  if (!manifests.has(pluginId)) throw new Error(`${pluginId}: mapping has no manifest`)
  const sourceDir = path.resolve(backendRoot, packagePath)
  if (!fs.statSync(sourceDir).isDirectory()) throw new Error(`${pluginId}: source is not a directory`)
}
for (const pluginId of manifests.keys()) {
  if (excludedManifests.has(pluginId)) continue
  if (!mappings.has(pluginId)) throw new Error(`${pluginId}: manifest has no source mapping`)
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

const entrypoint = readRepoFile('docker/entrypoint.sh')
expectText(entrypoint, 'cp -a bundled-plugins/. plugins/', 'docker/entrypoint.sh')

const pingbotInstall = readRepoFile('pingbot/deploy/systemd/install.sh')
expectText(pingbotInstall, '[[ "$CONFIG_EXPLICIT" == true || ! -e /etc/pingbot/config.yaml ]]', 'pingbot install.sh')
expectText(pingbotInstall, 'systemctl restart pingbot.service', 'pingbot install.sh')

const eslintConfig = readRepoFile('frontend/eslint.config.mjs')
expectText(eslintConfig, "'.vite-cache/**'", 'eslint.config.mjs')

console.log(`release smoke: ${mappings.size} plugin mappings cover ${manifests.size} manifests`)

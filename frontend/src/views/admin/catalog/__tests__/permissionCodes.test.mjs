import assert from 'node:assert/strict'
import { readFileSync } from 'node:fs'
import test from 'node:test'

const catalogView = readFileSync(new URL('../index.vue', import.meta.url), 'utf8')

test('region active switch uses the permission inferred by the backend route', () => {
  assert.match(catalogView, /hasPermission\('region\.active'\)/)
  assert.doesNotMatch(catalogView, /hasPermission\('region\.set_active'\)/)
})

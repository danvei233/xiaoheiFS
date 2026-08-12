import assert from 'node:assert/strict'
import { readFileSync } from 'node:fs'
import test from 'node:test'

const source = readFileSync(new URL('../index.vue', import.meta.url), 'utf8')

test('VPS table forwards every filter and consumes the filtered server total', () => {
  for (const parameter of ['status', 'keyword', 'region', 'expire_days', 'limit', 'offset']) {
    assert.match(source, new RegExp(`\\b${parameter}:`), `${parameter} must be sent to listVps`)
  }

  assert.match(source, /const serverTotal = Number\(response\.data\?\.total\)/)
  assert.match(source, /records: rows/)
  assert.match(source, /total: serverTotal/)
  assert.doesNotMatch(source, /const filterRows\s*=/)
  assert.doesNotMatch(source, /filtered\.slice\(/)
})

test('VPS table displays the same canonical status used by the backend filter', () => {
  assert.match(source, /String\(row\.status \?\? row\.Status \?\? 'unknown'\)/)
  assert.doesNotMatch(source, /statusFromAutomation/)
})

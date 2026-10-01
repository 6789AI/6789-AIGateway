import assert from 'node:assert/strict'
import { describe, test } from 'node:test'

import {
  gatewayRateLimitSchema,
  parseGatewayRateLimits,
} from '../gateway-rate-limit-schema'

describe('website request limit validation', () => {
  test('preserves the existing sensitive bucket until separate limits are enabled', () => {
    const defaults = parseGatewayRateLimits('')
    assert.equal(defaults.critical.limit, 20)
    assert.equal(defaults.login.enabled, false)
    assert.equal(defaults.session.enabled, false)
    assert.equal(defaults.session.limit, 120)
  })

  test('rejects invalid window and request limits before saving', () => {
    const defaults = parseGatewayRateLimits('')
    assert.equal(
      gatewayRateLimitSchema.safeParse({
        ...defaults,
        login: { enabled: true, limit: 0, window_seconds: 1200 },
      }).success,
      false
    )
    assert.equal(
      gatewayRateLimitSchema.safeParse({
        ...defaults,
        session: { enabled: true, limit: 120, window_seconds: 1201 },
      }).success,
      false
    )
  })

  test('parses a saved configuration for form reload', () => {
    const defaults = parseGatewayRateLimits('')
    const saved = {
      ...defaults,
      api: { enabled: true, limit: 800, window_seconds: 180 },
      login: { enabled: true, limit: 150, window_seconds: 1200 },
    }
    assert.deepEqual(parseGatewayRateLimits(JSON.stringify(saved)), saved)
  })
})

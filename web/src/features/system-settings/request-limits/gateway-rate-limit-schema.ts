/*
Copyright (C) 2023-2026 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the GNU
Affero General Public License for more details.

You should have received a copy of the GNU Affero General Public License
along with this program. If not, see <https://www.gnu.org/licenses/>.
*/
import { z } from 'zod'

const ruleSchema = z.object({
  enabled: z.boolean(),
  limit: z.number().int().min(1).max(1_000_000),
  window_seconds: z.number().int().min(1).max(1200),
})

export const gatewayRateLimitSchema = z.object({
  api: ruleSchema,
  web: ruleSchema,
  critical: ruleSchema,
  login: ruleSchema,
  session: ruleSchema,
})

export type GatewayRateLimits = z.infer<typeof gatewayRateLimitSchema>

const fallback: GatewayRateLimits = {
  api: { enabled: true, limit: 360, window_seconds: 180 },
  web: { enabled: true, limit: 120, window_seconds: 180 },
  critical: { enabled: true, limit: 20, window_seconds: 1200 },
  login: { enabled: false, limit: 20, window_seconds: 1200 },
  session: { enabled: false, limit: 120, window_seconds: 1200 },
}

export function parseGatewayRateLimits(value: string): GatewayRateLimits {
  if (!value) return gatewayRateLimitSchema.parse(fallback)
  try {
    return gatewayRateLimitSchema.parse(JSON.parse(value))
  } catch {
    return gatewayRateLimitSchema.parse(fallback)
  }
}

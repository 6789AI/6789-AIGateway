/*
Copyright (C) 2023-2026 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
GNU Affero General Public License for more details.

You should have received a copy of the GNU Affero General Public License
along with this program. If not, see <https://www.gnu.org/licenses/>.

For commercial licensing, please contact support@quantumnous.com
*/
export const NAV_COLOR_PATTERN = /^#[0-9a-fA-F]{6}$/

export function normalizeNavColor(value: unknown): string | undefined {
  if (typeof value !== 'string') return undefined
  const color = value.trim()
  return NAV_COLOR_PATTERN.test(color) ? color : undefined
}

export type SidebarNavColors = Record<string, string>

export function parseSidebarNavColors(raw: unknown): SidebarNavColors {
  let parsed: unknown = raw
  if (typeof raw === 'string') {
    try {
      parsed = JSON.parse(raw)
    } catch {
      return {}
    }
  }
  if (!parsed || typeof parsed !== 'object' || Array.isArray(parsed)) return {}

  const colors: SidebarNavColors = {}
  for (const [key, value] of Object.entries(parsed).slice(0, 100)) {
    if (key !== 'chat-presets' && !/^\/[a-zA-Z0-9/_-]{1,120}$/.test(key)) {
      continue
    }
    const color = normalizeNavColor(value)
    if (color) colors[key] = color
  }
  return colors
}

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
export const VIDEO_RESOLUTIONS = [
  '480p',
  '720p',
  '768p',
  '1080p',
  '2k',
] as const

export const isVideoResolutionPricesMap = (value: unknown): boolean => {
  if (!value || typeof value !== 'object' || Array.isArray(value)) return false
  return Object.entries(value).every(
    ([model, prices]) =>
      model.trim().length > 0 &&
      prices !== null &&
      typeof prices === 'object' &&
      !Array.isArray(prices) &&
      Object.entries(prices).every(
        ([resolution, price]) =>
          VIDEO_RESOLUTIONS.some((tier) => tier === resolution) &&
          typeof price === 'number' &&
          Number.isFinite(price) &&
          price >= 0
      )
  )
}

export const parseVideoResolutionPriceDraft = (
  draft: Record<string, string>
): Record<string, number> | null => {
  const prices: Record<string, number> = {}
  for (const [resolution, value] of Object.entries(draft)) {
    if (value.trim() === '') continue
    if (
      !/^(\d+(\.\d*)?|\.\d+)$/.test(value) ||
      !Number.isFinite(Number(value))
    ) {
      return null
    }
    prices[resolution] = Number(value)
  }
  return isVideoResolutionPricesMap({ model: prices }) ? prices : null
}

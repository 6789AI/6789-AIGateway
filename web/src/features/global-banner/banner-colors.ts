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
const hexColorPattern = /^#[0-9a-fA-F]{6}$/

export function safeHexColor(value: string, fallback: string): string {
  return hexColorPattern.test(value) ? value : fallback
}

export function getReadableTextColor(backgroundColor: string): string {
  const red = Number.parseInt(backgroundColor.slice(1, 3), 16)
  const green = Number.parseInt(backgroundColor.slice(3, 5), 16)
  const blue = Number.parseInt(backgroundColor.slice(5, 7), 16)
  const channels = [red, green, blue].map((channel) => {
    const normalized = channel / 255
    return normalized <= 0.04045
      ? normalized / 12.92
      : ((normalized + 0.055) / 1.055) ** 2.4
  })
  const luminance =
    channels[0] * 0.2126 + channels[1] * 0.7152 + channels[2] * 0.0722
  const blackContrast = (luminance + 0.05) / 0.05
  const whiteContrast = 1.05 / (luminance + 0.05)

  return blackContrast >= whiteContrast ? '#000000' : '#FFFFFF'
}

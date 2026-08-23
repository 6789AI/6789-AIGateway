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
import type { BannerIconName } from './marketing-banner-icons'

export type FreeModelPromotion = {
  model_name: string
  ends_at?: number
}

export type BannerConfig = {
  enabled: boolean
  content: string
  background_color: string
  text_color: string
  icon: BannerIconName
  countdown_enabled: boolean
  countdown_end_at: number
  link_url: string
}

export type FreeModelBannerStyle = Pick<
  BannerConfig,
  'background_color' | 'text_color' | 'icon' | 'link_url'
>

export type GlobalBannerData = {
  active: boolean
  global_banner?: BannerConfig
  free_model_banner: FreeModelBannerStyle
  server_time: number
  next_change_at?: number
  models: FreeModelPromotion[]
}

export type GlobalBannerResponse = {
  success: boolean
  data: GlobalBannerData
}

export type GlobalBannerView = {
  visible: boolean
  globalBanner: BannerConfig
  freeModelBanner: FreeModelBannerStyle
  models: FreeModelPromotion[]
  globalRemainingSeconds: number | null
  promotionRemainingSeconds: number | null
}

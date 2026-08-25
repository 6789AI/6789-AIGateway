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
export type ModelPromotion = {
  model_name: string
  promotion_type: 'free' | 'discount' | 'fixed_price'
  price?: number
  discount_rate?: number
  ends_at?: number
}

export type BannerConfig = {
  enabled: boolean
  content: string
  background_color: string
  text_color: string
  icon: string
  countdown_enabled: boolean
  countdown_end_at: number
  link_url: string
  button_text: string
  button_color: string
}

export type FreeModelBannerStyle = Pick<
  BannerConfig,
  | 'background_color'
  | 'text_color'
  | 'icon'
  | 'link_url'
  | 'button_text'
  | 'button_color'
>

export type GlobalBannerData = {
  active: boolean
  global_banner?: BannerConfig
  free_model_banner: FreeModelBannerStyle
  server_time: number
  next_change_at?: number
  models?: ModelPromotion[] | null
}

export type GlobalBannerResponse = {
  success: boolean
  data: GlobalBannerData
}

export type GlobalBannerView = {
  visible: boolean
  globalBanner: BannerConfig
  freeModelBanner: FreeModelBannerStyle
  models: ModelPromotion[]
  globalRemainingSeconds: number | null
  promotionRemainingSeconds: number | null
}

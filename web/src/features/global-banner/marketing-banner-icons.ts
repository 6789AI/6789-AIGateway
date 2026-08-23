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
import {
  CouponPercentIcon,
  CrownIcon,
  DiscountTag02Icon,
  FireIcon,
  FlashIcon,
  GiftIcon,
  HeartIcon,
  Megaphone01Icon,
  Notification02Icon,
  PartyIcon,
  Rocket01Icon,
  SparklesIcon,
  StarIcon,
} from '@hugeicons/core-free-icons'

export const MARKETING_BANNER_ICON_NAMES = [
  'megaphone',
  'gift',
  'sparkles',
  'rocket',
  'bell',
  'coupon',
  'discount',
  'fire',
  'star',
  'crown',
  'party',
  'lightning',
  'heart',
] as const

export type MarketingBannerIconName =
  (typeof MARKETING_BANNER_ICON_NAMES)[number]

export const MARKETING_BANNER_ICON_OPTIONS = [
  { value: 'megaphone', labelKey: 'Megaphone', icon: Megaphone01Icon },
  { value: 'gift', labelKey: 'Gift', icon: GiftIcon },
  { value: 'sparkles', labelKey: 'Sparkles', icon: SparklesIcon },
  { value: 'rocket', labelKey: 'Rocket', icon: Rocket01Icon },
  { value: 'bell', labelKey: 'Bell', icon: Notification02Icon },
  { value: 'coupon', labelKey: 'Coupon', icon: CouponPercentIcon },
  { value: 'discount', labelKey: 'Discount', icon: DiscountTag02Icon },
  { value: 'fire', labelKey: 'Fire', icon: FireIcon },
  { value: 'star', labelKey: 'Star', icon: StarIcon },
  { value: 'crown', labelKey: 'Crown', icon: CrownIcon },
  { value: 'party', labelKey: 'Celebration', icon: PartyIcon },
  { value: 'lightning', labelKey: 'Lightning', icon: FlashIcon },
  { value: 'heart', labelKey: 'Heart', icon: HeartIcon },
] as const satisfies ReadonlyArray<{
  value: MarketingBannerIconName
  labelKey: string
  icon: typeof Megaphone01Icon
}>

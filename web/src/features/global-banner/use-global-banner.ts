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
import { useQuery } from '@tanstack/react-query'
import { useEffect, useState } from 'react'

import { getGlobalBanner } from './api'
import type {
  BannerConfig,
  FreeModelBannerStyle,
  GlobalBannerData,
  GlobalBannerView,
  ModelPromotion,
} from './types'

const BANNER_REFRESH_INTERVAL = 15_000
const globalBannerQueryOptions = {
  queryKey: ['global-banner'] as const,
  queryFn: getGlobalBanner,
  staleTime: BANNER_REFRESH_INTERVAL,
  refetchInterval: BANNER_REFRESH_INTERVAL,
  refetchIntervalInBackground: false,
}
const defaultGlobalBanner: BannerConfig = {
  enabled: false,
  content: '',
  background_color: '#0EA5E9',
  text_color: '#082F49',
  icon: 'gift',
  countdown_enabled: false,
  countdown_end_at: 0,
  link_url: '/pricing',
  button_text: '',
  button_color: '#FFFFFF',
}

const defaultFreeModelBanner: FreeModelBannerStyle = {
  background_color: '#A3E635',
  text_color: '#1A2E05',
  icon: 'megaphone',
  link_url: '',
  button_text: '',
  button_color: '#FFFFFF',
}

const inactiveBannerView: GlobalBannerView = {
  visible: false,
  globalBanner: defaultGlobalBanner,
  freeModelBanner: defaultFreeModelBanner,
  models: [],
  globalRemainingSeconds: null,
  promotionRemainingSeconds: null,
}

export function getGlobalBannerView(
  data: GlobalBannerData | undefined,
  dataUpdatedAt: number,
  now: number
): GlobalBannerView {
  if (!data || dataUpdatedAt <= 0) return inactiveBannerView

  const elapsedSeconds = Math.max(0, now - dataUpdatedAt) / 1000
  const serverTime = data.server_time + elapsedSeconds
  const models = (data.models ?? []).filter(
    (model) => !model.ends_at || model.ends_at > serverTime
  )
  const finiteEndTimes = models
    .map((model) => model.ends_at)
    .filter((endsAt): endsAt is number => !!endsAt && endsAt > serverTime)
  const nextChangeAt =
    finiteEndTimes.length > 0 ? Math.min(...finiteEndTimes) : null
  const globalBanner = {
    ...defaultGlobalBanner,
    ...data.global_banner,
  }
  const globalVisible =
    globalBanner.enabled &&
    globalBanner.content.trim().length > 0 &&
    (!globalBanner.countdown_enabled ||
      globalBanner.countdown_end_at > serverTime)
  const freeModelBanner = {
    ...defaultFreeModelBanner,
    ...data.free_model_banner,
  }

  return {
    visible: globalVisible || models.length > 0,
    globalBanner: {
      ...globalBanner,
      enabled: globalVisible,
    },
    freeModelBanner,
    models,
    globalRemainingSeconds:
      globalVisible && globalBanner.countdown_enabled
        ? Math.max(0, Math.ceil(globalBanner.countdown_end_at - serverTime))
        : null,
    promotionRemainingSeconds:
      nextChangeAt === null
        ? null
        : Math.max(0, Math.ceil(nextChangeAt - serverTime)),
  }
}

export function useGlobalBanner(): GlobalBannerView {
  const [now, setNow] = useState(() => Date.now())
  const query = useQuery(globalBannerQueryOptions)

  const data = query.data?.success ? query.data.data : undefined
  const hasLiveCountdown =
    data?.models?.some((model) => !!model.ends_at) ||
    (data?.global_banner?.enabled &&
      data.global_banner.countdown_enabled &&
      data.global_banner.countdown_end_at > 0)

  useEffect(() => {
    if (!hasLiveCountdown) return

    const intervalId = window.setInterval(() => setNow(Date.now()), 1000)
    return () => window.clearInterval(intervalId)
  }, [hasLiveCountdown, query.dataUpdatedAt])

  return getGlobalBannerView(data, query.dataUpdatedAt, now)
}

export function useActiveModelPromotions(): ModelPromotion[] {
  return useGlobalBanner().models
}

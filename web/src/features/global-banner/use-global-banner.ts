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
  GlobalBannerData,
  GlobalBannerView,
  MarketingBannerConfig,
} from './types'

const BANNER_REFRESH_INTERVAL = 15_000
const defaultMarketingBanner: MarketingBannerConfig = {
  enabled: false,
  content: '',
  background_color: '#A3E635',
  text_color: '#1A2E05',
  icon: 'megaphone',
  countdown_enabled: false,
  countdown_end_at: 0,
  link_url: '',
}

const inactiveBannerView: GlobalBannerView = {
  visible: false,
  marketingBanner: defaultMarketingBanner,
  models: [],
  marketingRemainingSeconds: null,
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
  const models = data.models.filter(
    (model) => !model.ends_at || model.ends_at > serverTime
  )
  const finiteEndTimes = models
    .map((model) => model.ends_at)
    .filter((endsAt): endsAt is number => !!endsAt && endsAt > serverTime)
  const nextChangeAt =
    finiteEndTimes.length > 0 ? Math.min(...finiteEndTimes) : null
  const marketingBanner = data.marketing_banner ?? defaultMarketingBanner
  const marketingVisible =
    marketingBanner.enabled &&
    marketingBanner.content.trim().length > 0 &&
    (!marketingBanner.countdown_enabled ||
      marketingBanner.countdown_end_at > serverTime)

  return {
    visible: marketingVisible || models.length > 0,
    marketingBanner: {
      ...marketingBanner,
      enabled: marketingVisible,
    },
    models,
    marketingRemainingSeconds:
      marketingVisible && marketingBanner.countdown_enabled
        ? Math.max(0, Math.ceil(marketingBanner.countdown_end_at - serverTime))
        : null,
    promotionRemainingSeconds:
      nextChangeAt === null
        ? null
        : Math.max(0, Math.ceil(nextChangeAt - serverTime)),
  }
}

export function useGlobalBanner(): GlobalBannerView {
  const [now, setNow] = useState(() => Date.now())
  const query = useQuery({
    queryKey: ['global-banner'],
    queryFn: getGlobalBanner,
    staleTime: BANNER_REFRESH_INTERVAL,
    refetchInterval: BANNER_REFRESH_INTERVAL,
    refetchIntervalInBackground: false,
  })

  const data = query.data?.success ? query.data.data : undefined
  const hasLiveCountdown =
    data?.models.some((model) => !!model.ends_at) ||
    (data?.marketing_banner.enabled &&
      data.marketing_banner.countdown_enabled &&
      data.marketing_banner.countdown_end_at > 0)

  useEffect(() => {
    setNow(Date.now())
    if (!hasLiveCountdown) return

    const intervalId = window.setInterval(() => setNow(Date.now()), 1000)
    return () => window.clearInterval(intervalId)
  }, [hasLiveCountdown, query.dataUpdatedAt])

  return getGlobalBannerView(data, query.dataUpdatedAt, now)
}

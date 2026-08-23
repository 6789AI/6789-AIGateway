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
import { ArrowRight01Icon, GiftIcon } from '@hugeicons/core-free-icons'
import { HugeiconsIcon } from '@hugeicons/react'
import { Link } from '@tanstack/react-router'
import { useEffect } from 'react'
import { useTranslation } from 'react-i18next'

import { getCountdownParts } from './countdown'
import { MarketingBannerIcon } from './marketing-banner-icon'
import type { GlobalBannerView } from './types'
import { useGlobalBanner } from './use-global-banner'

const bannerRowHeightRem = 2.5
const fallbackBackgroundColor = '#A3E635'
const fallbackTextColor = '#1A2E05'

type GlobalBannerProps = {
  view: GlobalBannerView
}

type GlobalBannerFrameProps = GlobalBannerProps & {
  children: React.ReactNode
}

function CountdownBlock(props: { value: string }) {
  return (
    <span className='inline-flex h-6 min-w-7 items-center justify-center rounded bg-white/90 px-1 font-mono text-xs font-semibold text-black/80 tabular-nums'>
      {props.value}
    </span>
  )
}

function Countdown(props: { remainingSeconds: number; label: string }) {
  const countdown = getCountdownParts(props.remainingSeconds)
  const countdownText = `${countdown.days}:${countdown.hours}:${countdown.minutes}:${countdown.seconds}`

  return (
    <time
      className='flex shrink-0 items-center gap-0.5'
      aria-label={`${props.label}: ${countdownText}`}
    >
      <span aria-hidden='true' className='flex items-center gap-0.5'>
        <CountdownBlock value={countdown.days} />
        <span className='text-xs font-semibold'>:</span>
        <CountdownBlock value={countdown.hours} />
        <span className='text-xs font-semibold'>:</span>
        <CountdownBlock value={countdown.minutes} />
        <span className='text-xs font-semibold'>:</span>
        <CountdownBlock value={countdown.seconds} />
      </span>
    </time>
  )
}

function safeHexColor(value: string, fallback: string): string {
  return /^#[0-9a-fA-F]{6}$/.test(value) ? value : fallback
}

function safeBannerLink(value: string): string | null {
  const trimmed = value.trim()
  if (/^\/(?!\/)/.test(trimmed)) return trimmed
  try {
    const parsed = new URL(trimmed)
    return parsed.protocol === 'http:' || parsed.protocol === 'https:'
      ? parsed.toString()
      : null
  } catch {
    return null
  }
}

function MarketingBannerRow(props: GlobalBannerProps) {
  const { t } = useTranslation()
  const link = safeBannerLink(props.view.marketingBanner.link_url)
  const content = (
    <div className='mx-auto flex h-full w-full max-w-7xl min-w-0 items-center justify-center gap-2 px-2 sm:px-4'>
      <MarketingBannerIcon
        name={props.view.marketingBanner.icon}
        className='size-4 shrink-0'
      />
      <span className='min-w-0 truncate text-xs font-semibold'>
        {props.view.marketingBanner.content}
      </span>
      {props.view.marketingRemainingSeconds !== null && (
        <Countdown
          remainingSeconds={props.view.marketingRemainingSeconds}
          label={t('Promotion ends in')}
        />
      )}
      {link && (
        <HugeiconsIcon
          icon={ArrowRight01Icon}
          strokeWidth={2}
          className='size-4 shrink-0'
          aria-hidden='true'
        />
      )}
    </div>
  )

  return (
    <aside
      className='flex h-10 w-full items-center'
      style={{
        backgroundColor: safeHexColor(
          props.view.marketingBanner.background_color,
          fallbackBackgroundColor
        ),
        color: safeHexColor(
          props.view.marketingBanner.text_color,
          fallbackTextColor
        ),
      }}
      aria-label={t('Marketing banner')}
      data-banner-kind='marketing'
    >
      {link ? (
        <a
          href={link}
          target={link.startsWith('/') ? undefined : '_blank'}
          rel='noreferrer'
          className='h-full w-full outline-none focus-visible:ring-2 focus-visible:ring-current focus-visible:ring-inset'
          aria-label={`${t('Open promotion')}: ${props.view.marketingBanner.content}`}
        >
          {content}
        </a>
      ) : (
        content
      )}
    </aside>
  )
}

function FreeModelBannerRow(props: GlobalBannerProps) {
  const { t } = useTranslation()
  const modelSummary = props.view.models
    .slice(0, 3)
    .map((model) => model.model_name)
    .join(', ')
  const hiddenModelCount = Math.max(0, props.view.models.length - 3)
  const desktopModelSummary = hiddenModelCount
    ? `${modelSummary} +${hiddenModelCount}`
    : modelSummary

  return (
    <aside
      className='bg-foreground text-background flex h-10 w-full items-center'
      aria-label={t('Limited-time free')}
      data-banner-kind='free-models'
    >
      <div className='mx-auto flex h-full w-full max-w-7xl min-w-0 items-center justify-center gap-2 px-2 sm:px-4'>
        <HugeiconsIcon
          icon={GiftIcon}
          strokeWidth={2}
          className='size-4 shrink-0'
          aria-hidden='true'
        />
        <span className='hidden min-w-0 truncate text-xs font-semibold sm:inline'>
          {t('{{models}} are free now', {
            models: desktopModelSummary,
          })}
        </span>

        {props.view.promotionRemainingSeconds !== null ? (
          <Countdown
            remainingSeconds={props.view.promotionRemainingSeconds}
            label={t('Free period remaining')}
          />
        ) : (
          <span className='bg-background/90 text-foreground shrink-0 rounded px-2 py-1 text-xs font-semibold'>
            {t('Free now')}
          </span>
        )}

        <Link
          to='/pricing'
          className='inline-flex size-7 shrink-0 items-center justify-center rounded outline-none hover:bg-white/20 focus-visible:ring-2 focus-visible:ring-current'
          aria-label={t('View models')}
          title={t('View models')}
        >
          <HugeiconsIcon
            icon={ArrowRight01Icon}
            strokeWidth={2}
            aria-hidden='true'
          />
        </Link>
      </div>
    </aside>
  )
}

export function GlobalBanner(props: GlobalBannerProps) {
  return (
    <div className='flex w-full flex-col'>
      {props.view.marketingBanner.enabled && (
        <MarketingBannerRow view={props.view} />
      )}
      {props.view.models.length > 0 && <FreeModelBannerRow view={props.view} />}
    </div>
  )
}

export function GlobalBannerFrame(props: GlobalBannerFrameProps) {
  const bannerCount =
    Number(props.view.marketingBanner.enabled) +
    Number(props.view.models.length > 0)
  const bannerHeight = `${bannerCount * bannerRowHeightRem}rem`

  useEffect(() => {
    const root = document.documentElement
    const previousHeight = root.style.getPropertyValue('--global-banner-height')
    root.style.setProperty('--global-banner-height', bannerHeight)
    return () => {
      if (previousHeight) {
        root.style.setProperty('--global-banner-height', previousHeight)
      } else {
        root.style.removeProperty('--global-banner-height')
      }
    }
  }, [bannerHeight])

  return (
    <div
      className='min-h-svh'
      style={
        {
          '--global-banner-height': bannerHeight,
          paddingTop: bannerHeight,
        } as React.CSSProperties
      }
      data-global-banner-visible={props.view.visible}
      data-global-banner-count={bannerCount}
    >
      {props.view.visible && (
        <div className='fixed inset-x-0 top-0 z-[100]'>
          <GlobalBanner view={props.view} />
        </div>
      )}
      {props.children}
    </div>
  )
}

export function GlobalBannerHost(props: { children: React.ReactNode }) {
  const view = useGlobalBanner()
  return <GlobalBannerFrame view={view}>{props.children}</GlobalBannerFrame>
}

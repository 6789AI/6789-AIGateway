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
import { useEffect } from 'react'
import { useTranslation } from 'react-i18next'

import { buttonVariants } from '@/components/ui/button'
import { toIntlLocale } from '@/i18n/languages'

import { getReadableTextColor, safeHexColor } from './banner-colors'
import { getCountdownParts } from './countdown'
import { MarketingBannerIcon } from './marketing-banner-icon'
import type { BannerConfig, GlobalBannerView } from './types'
import { useGlobalBanner } from './use-global-banner'

const bannerRowHeightRem = 2.5
const fallbackBackgroundColor = '#A3E635'
const fallbackTextColor = '#1A2E05'
const fallbackButtonColor = '#FFFFFF'

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

type ConfiguredBannerRowProps = {
  config: BannerConfig
  remainingSeconds: number | null
  kind: 'global'
  ariaLabel: string
  countdownLabel: string
  openLabel: string
  defaultButtonText: string
}

function ConfiguredBannerRow(props: ConfiguredBannerRowProps) {
  const link = safeBannerLink(props.config.link_url)
  const buttonText = props.config.button_text.trim() || props.defaultButtonText
  const buttonColor = safeHexColor(
    props.config.button_color,
    fallbackButtonColor
  )

  return (
    <aside
      className='flex h-10 w-full items-center'
      style={{
        backgroundColor: safeHexColor(
          props.config.background_color,
          fallbackBackgroundColor
        ),
        color: safeHexColor(props.config.text_color, fallbackTextColor),
      }}
      aria-label={props.ariaLabel}
      data-banner-kind={props.kind}
    >
      <div className='mx-auto flex h-full w-full max-w-7xl min-w-0 items-center justify-center gap-2 px-2 sm:px-4'>
        <MarketingBannerIcon
          name={props.config.icon}
          className='size-4 shrink-0'
        />
        <span className='min-w-0 truncate text-xs font-semibold'>
          {props.config.content}
        </span>
        {props.remainingSeconds !== null && (
          <Countdown
            remainingSeconds={props.remainingSeconds}
            label={props.countdownLabel}
          />
        )}
        {link && (
          <a
            href={link}
            target={link.startsWith('/') ? undefined : '_blank'}
            rel='noreferrer'
            aria-label={`${props.openLabel}: ${props.config.content}`}
            className={buttonVariants({
              size: 'xs',
              className: 'max-w-20 px-2 hover:opacity-90 sm:max-w-36',
            })}
            style={{
              backgroundColor: buttonColor,
              color: getReadableTextColor(buttonColor),
            }}
          >
            <span className='truncate'>{buttonText}</span>
          </a>
        )}
      </div>
    </aside>
  )
}

function GlobalAnnouncementBannerRow(props: GlobalBannerProps) {
  const { t } = useTranslation()
  return (
    <ConfiguredBannerRow
      config={props.view.globalBanner}
      remainingSeconds={props.view.globalRemainingSeconds}
      kind='global'
      ariaLabel={t('Global announcement banner')}
      countdownLabel={t('Announcement ends in')}
      openLabel={t('Open announcement')}
      defaultButtonText={t('Click to view')}
    />
  )
}

function ModelPromotionBannerRow(props: GlobalBannerProps) {
  const { t, i18n } = useTranslation()
  const configuredLink = safeBannerLink(props.view.freeModelBanner.link_url)
  const link = configuredLink ?? '/pricing'
  const buttonText =
    props.view.freeModelBanner.button_text.trim() || t('Click to view')
  const buttonColor = safeHexColor(
    props.view.freeModelBanner.button_color,
    fallbackButtonColor
  )
  const numberFormatter = new Intl.NumberFormat(
    toIntlLocale(i18n.resolvedLanguage || i18n.language),
    { maximumFractionDigits: 6 }
  )
  const promotionSummary = props.view.models
    .slice(0, 3)
    .map((promotion) => {
      if (promotion.promotion_type === 'free') {
        return t('{{model}} is free now', { model: promotion.model_name })
      }
      if (
        promotion.promotion_type === 'discount' &&
        promotion.discount_rate !== undefined
      ) {
        const percent = numberFormatter.format(
          Math.max(0, (1 - promotion.discount_rate) * 100)
        )
        return t('{{model}} is {{percent}}% off', {
          model: promotion.model_name,
          percent,
        })
      }
      return t('{{model}} is ${{price}} per request', {
        model: promotion.model_name,
        price: numberFormatter.format(promotion.price ?? 0),
      })
    })
    .join(' · ')
  const hiddenModelCount = Math.max(0, props.view.models.length - 3)
  const visibleSummary = hiddenModelCount
    ? `${promotionSummary} +${hiddenModelCount}`
    : promotionSummary

  return (
    <aside
      className='flex h-10 w-full items-center'
      style={{
        backgroundColor: safeHexColor(
          props.view.freeModelBanner.background_color,
          fallbackBackgroundColor
        ),
        color: safeHexColor(
          props.view.freeModelBanner.text_color,
          fallbackTextColor
        ),
      }}
      aria-label={t('Limited-time model offer')}
      data-banner-kind='model-promotions'
    >
      <div className='mx-auto flex h-full w-full max-w-7xl min-w-0 items-center justify-center gap-2 px-2 sm:px-4'>
        <MarketingBannerIcon
          name={props.view.freeModelBanner.icon}
          className='size-4 shrink-0'
        />
        <span className='min-w-0 truncate text-xs font-semibold'>
          {visibleSummary}
        </span>

        {props.view.promotionRemainingSeconds !== null ? (
          <Countdown
            remainingSeconds={props.view.promotionRemainingSeconds}
            label={t('Offer period remaining')}
          />
        ) : (
          <span className='bg-background/90 text-foreground shrink-0 rounded px-2 py-1 text-xs font-semibold'>
            {t('Active now')}
          </span>
        )}

        <a
          href={link}
          target={link.startsWith('/') ? undefined : '_blank'}
          rel='noreferrer'
          aria-label={buttonText}
          className={buttonVariants({
            size: 'xs',
            className: 'max-w-20 px-2 hover:opacity-90 sm:max-w-36',
          })}
          style={{
            backgroundColor: buttonColor,
            color: getReadableTextColor(buttonColor),
          }}
        >
          <span className='truncate'>{buttonText}</span>
        </a>
      </div>
    </aside>
  )
}

export function GlobalBanner(props: GlobalBannerProps) {
  return (
    <div className='flex w-full flex-col'>
      {props.view.globalBanner.enabled && (
        <GlobalAnnouncementBannerRow view={props.view} />
      )}
      {props.view.models.length > 0 && (
        <ModelPromotionBannerRow view={props.view} />
      )}
    </div>
  )
}

export function GlobalBannerFrame(props: GlobalBannerFrameProps) {
  const bannerCount =
    Number(props.view.globalBanner.enabled) +
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

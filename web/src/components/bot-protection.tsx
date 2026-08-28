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
import { useEffect, useRef } from 'react'
import { useTranslation } from 'react-i18next'

export type BotProtectionProvider =
  | 'turnstile'
  | 'recaptcha'
  | 'geetest_v4'
  | 'cap'

type GeeTestValidation = {
  lot_number: string
  captcha_output: string
  pass_token: string
  gen_time: string
}

type GeeTestInstance = {
  appendTo: (element: HTMLElement) => void
  destroy?: () => void
  getValidate: () => GeeTestValidation | null
  onSuccess: (callback: () => void) => GeeTestInstance
  onError: (callback: () => void) => GeeTestInstance
  onClose: (callback: () => void) => GeeTestInstance
}

declare global {
  interface Window {
    turnstile?: {
      render: (element: HTMLElement, options: Record<string, unknown>) => string
      remove?: (widgetId: string) => void
    }
    grecaptcha?: {
      render: (element: HTMLElement, options: Record<string, unknown>) => number
      reset?: (widgetId: number) => void
    }
    initGeetest4?: (
      options: Record<string, unknown>,
      callback: (instance: GeeTestInstance) => void
    ) => void
    CAP_CUSTOM_WASM_URL?: string
  }
}

type BotProtectionWidgetProps = {
  provider: BotProtectionProvider
  siteKey?: string
  capAPIEndpoint?: string
  onVerify: (proof: string) => void
  onExpire?: () => void
  className?: string
}

const scriptPromises = new Map<string, Promise<void>>()
const capWasmURL = new URL(
  '@cap.js/wasm/browser/cap_wasm_bg.wasm',
  import.meta.url
).href

function loadExternalScript(id: string, source: string): Promise<void> {
  const pending = scriptPromises.get(id)
  if (pending) return pending

  const promise = new Promise<void>((resolve, reject) => {
    const existing = document.querySelector<HTMLScriptElement>(`script#${id}`)
    const script = existing ?? document.createElement('script')
    const handleLoad = () => resolve()
    const handleError = () => {
      script.remove()
      reject(new Error(`Failed to load ${source}`))
    }

    script.addEventListener('load', handleLoad, { once: true })
    script.addEventListener('error', handleError, { once: true })
    if (!existing) {
      script.id = id
      script.src = source
      script.async = true
      script.defer = true
      document.head.appendChild(script)
    }
  })
  scriptPromises.set(id, promise)
  promise.catch(() => scriptPromises.delete(id))
  return promise
}

export function BotProtectionWidget(props: BotProtectionWidgetProps) {
  const { t } = useTranslation()
  const containerRef = useRef<HTMLDivElement | null>(null)
  const onVerifyRef = useRef(props.onVerify)
  const onExpireRef = useRef(props.onExpire)

  useEffect(() => {
    onVerifyRef.current = props.onVerify
    onExpireRef.current = props.onExpire
  }, [props.onExpire, props.onVerify])

  useEffect(() => {
    const container = containerRef.current
    if (!container) return
    const widgetContainer: HTMLDivElement = container
    let cancelled = false
    let cleanup = () => {}

    const reportExpired = () => onExpireRef.current?.()

    async function renderWidget() {
      if (props.provider === 'turnstile') {
        if (!props.siteKey) return
        if (!window.turnstile) {
          await loadExternalScript(
            'cf-turnstile',
            'https://challenges.cloudflare.com/turnstile/v0/api.js?render=explicit'
          )
        }
        if (cancelled || !window.turnstile) return
        const widgetId = window.turnstile.render(widgetContainer, {
          sitekey: props.siteKey,
          callback: (token: string) => onVerifyRef.current(token),
          'error-callback': reportExpired,
          'expired-callback': reportExpired,
        })
        cleanup = () => window.turnstile?.remove?.(widgetId)
        return
      }

      if (props.provider === 'recaptcha') {
        if (!props.siteKey) return
        if (!window.grecaptcha) {
          await loadExternalScript(
            'google-recaptcha',
            'https://www.recaptcha.net/recaptcha/api.js?render=explicit'
          )
        }
        if (cancelled || !window.grecaptcha) return
        const widgetId = window.grecaptcha.render(widgetContainer, {
          sitekey: props.siteKey,
          callback: (token: string) => onVerifyRef.current(token),
          'error-callback': reportExpired,
          'expired-callback': reportExpired,
        })
        cleanup = () => window.grecaptcha?.reset?.(widgetId)
        return
      }

      if (props.provider === 'geetest_v4') {
        if (!props.siteKey) return
        if (!window.initGeetest4) {
          await loadExternalScript(
            'geetest-v4',
            'https://static.geetest.com/v4/gt4.js'
          )
        }
        if (cancelled || !window.initGeetest4) return
        window.initGeetest4(
          {
            captchaId: props.siteKey,
            product: 'float',
            protocol: 'https://',
          },
          (instance) => {
            if (cancelled) {
              instance.destroy?.()
              return
            }
            instance
              .onSuccess(() => {
                const validation = instance.getValidate()
                if (validation) {
                  onVerifyRef.current(JSON.stringify(validation))
                }
              })
              .onError(reportExpired)
              .onClose(reportExpired)
            instance.appendTo(widgetContainer)
            cleanup = () => instance.destroy?.()
          }
        )
        return
      }

      if (!props.capAPIEndpoint) return
      window.CAP_CUSTOM_WASM_URL = capWasmURL
      await import('cap-widget')
      if (cancelled) return

      const widget = document.createElement('cap-widget')
      widget.setAttribute('data-cap-api-endpoint', props.capAPIEndpoint)
      widget.setAttribute('data-cap-i18n-initial-state', t("I'm human"))
      widget.setAttribute('data-cap-i18n-verifying-label', t('Verifying...'))
      widget.setAttribute('data-cap-i18n-solved-label', t("You're human"))
      widget.setAttribute('data-cap-i18n-error-label', t('Verification failed'))
      widget.setAttribute(
        'data-cap-i18n-required-label',
        t('Please complete the human verification')
      )
      const handleSolve = (event: Event) => {
        const token = (event as CustomEvent<{ token?: string }>).detail?.token
        if (token) onVerifyRef.current(token)
      }
      widget.addEventListener('solve', handleSolve)
      widget.addEventListener('error', reportExpired)
      widgetContainer.appendChild(widget)
      cleanup = () => {
        widget.removeEventListener('solve', handleSolve)
        widget.removeEventListener('error', reportExpired)
        widget.remove()
      }
    }

    renderWidget().catch(reportExpired)
    return () => {
      cancelled = true
      cleanup()
      widgetContainer.replaceChildren()
    }
  }, [
    props.capAPIEndpoint,
    props.provider,
    props.siteKey,
    t,
  ])

  return <div ref={containerRef} className={props.className} />
}

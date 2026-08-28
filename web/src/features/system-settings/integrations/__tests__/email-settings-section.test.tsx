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
import assert from 'node:assert/strict'
import { after, describe, test } from 'node:test'

import { Window } from 'happy-dom'

const domWindow = new Window({ url: 'http://localhost/' })
const domGlobals = [
  'window',
  'document',
  'navigator',
  'HTMLElement',
  'HTMLInputElement',
  'SVGElement',
  'Node',
  'Element',
  'Event',
  'CustomEvent',
  'MutationObserver',
  'requestAnimationFrame',
  'cancelAnimationFrame',
  'getComputedStyle',
] as const

for (const key of domGlobals) {
  Object.defineProperty(globalThis, key, {
    configurable: true,
    value: domWindow[key],
  })
}

const { act } = await import('react')
const { createRoot } = await import('react-dom/client')
const { QueryClient, QueryClientProvider } =
  await import('@tanstack/react-query')
const { createInstance } = await import('i18next')
const { I18nextProvider, initReactI18next } = await import('react-i18next')
const { SettingsPageProvider } =
  await import('../../components/settings-page-context')
const { EmailSettingsSection } = await import('../email-settings-section')

const i18n = createInstance()
await i18n.use(initReactI18next).init({
  lng: 'en',
  resources: { en: { translation: {} } },
})

const reactTestGlobals = globalThis as typeof globalThis & {
  IS_REACT_ACT_ENVIRONMENT?: boolean
}
reactTestGlobals.IS_REACT_ACT_ENVIRONMENT = true

const defaultValues = {
  EmailSenderProvider: 'smtp' as const,
  SMTPServer: 'smtp.example.com',
  SMTPPort: '587',
  SMTPAccount: 'smtp-user',
  SMTPFromName: 'SMTP Sender',
  SMTPFrom: 'smtp@example.com',
  SMTPToken: '',
  SMTPSSLEnabled: false,
  SMTPStartTLSEnabled: true,
  SMTPInsecureSkipVerify: false,
  SMTPForceAuthLogin: false,
  BTMailAPIURL:
    'https://panel.example.com:8888/mail_sys/send_mail_http.json',
  BTMailFromName: 'BT Sender',
  BTMailFrom: 'bt@example.com',
  BTMailPassword: '',
}

describe('sending settings provider switch', () => {
  after(() => domWindow.close())

  test('retains each provider configuration while switching', async () => {
    const container = document.createElement('div')
    const actionsContainer = document.createElement('div')
    document.body.append(container, actionsContainer)
    const root = createRoot(container)
    const queryClient = new QueryClient({
      defaultOptions: { mutations: { retry: false } },
    })

    await act(async () => {
      root.render(
        <QueryClientProvider client={queryClient}>
          <I18nextProvider i18n={i18n}>
            <SettingsPageProvider actionsContainer={actionsContainer}>
              <EmailSettingsSection
                defaultValues={defaultValues}
                btMailPasswordConfigured
              />
            </SettingsPageProvider>
          </I18nextProvider>
        </QueryClientProvider>
      )
    })

    const smtpInput = container.querySelector<HTMLInputElement>(
      'input[name="SMTPServer"]'
    )
    assert.equal(smtpInput?.value, 'smtp.example.com')

    const btMailButton = [...container.querySelectorAll('button')].find(
      (button) => button.textContent === 'BT Mail API'
    )
    assert.ok(btMailButton)
    await act(async () => {
      btMailButton.dispatchEvent(
        new domWindow.MouseEvent('click', { bubbles: true }) as unknown as Event
      )
    })

    const btMailURLInput = container.querySelector<HTMLInputElement>(
      'input[name="BTMailAPIURL"]'
    )
    assert.equal(
      btMailURLInput?.value,
      'https://panel.example.com:8888/mail_sys/send_mail_http.json'
    )
    assert.equal(
      container.querySelector<HTMLInputElement>('input[name="BTMailFromName"]')
        ?.value,
      'BT Sender'
    )
    const btMailNameInput = container.querySelector<HTMLInputElement>(
      'input[name="BTMailFromName"]'
    )
    assert.ok(btMailNameInput)
    await act(async () => {
      const valueSetter = Object.getOwnPropertyDescriptor(
        domWindow.HTMLInputElement.prototype,
        'value'
      )?.set
      valueSetter?.call(btMailNameInput, 'Updated BT Sender')
      btMailNameInput.dispatchEvent(
        new domWindow.Event('input', { bubbles: true }) as unknown as Event
      )
    })
    assert.ok(btMailURLInput)
    await act(async () => {
      const valueSetter = Object.getOwnPropertyDescriptor(
        domWindow.HTMLInputElement.prototype,
        'value'
      )?.set
      valueSetter?.call(
        btMailURLInput,
        'https://new-panel.example.com/mail_sys/send_mail_http.json'
      )
      btMailURLInput.dispatchEvent(
        new domWindow.Event('input', { bubbles: true }) as unknown as Event
      )
    })

    const smtpButton = [...container.querySelectorAll('button')].find(
      (button) => button.textContent === 'SMTP'
    )
    assert.ok(smtpButton)
    await act(async () => {
      smtpButton.dispatchEvent(
        new domWindow.MouseEvent('click', { bubbles: true }) as unknown as Event
      )
    })
    assert.equal(
      container.querySelector<HTMLInputElement>('input[name="SMTPServer"]')
        ?.value,
      'smtp.example.com'
    )

    await act(async () => {
      btMailButton.dispatchEvent(
        new domWindow.MouseEvent('click', { bubbles: true }) as unknown as Event
      )
    })
    assert.equal(
      container.querySelector<HTMLInputElement>('input[name="BTMailAPIURL"]')
        ?.value,
      'https://new-panel.example.com/mail_sys/send_mail_http.json'
    )
    assert.equal(
      container.querySelector<HTMLInputElement>('input[name="BTMailFromName"]')
        ?.value,
      'Updated BT Sender'
    )

    await act(async () => root.unmount())
    queryClient.clear()
    container.remove()
    actionsContainer.remove()
  })
})

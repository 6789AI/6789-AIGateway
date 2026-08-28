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
import type { UseFormReturn } from 'react-hook-form'
import { useTranslation } from 'react-i18next'

import {
  FormControl,
  FormDescription,
  FormField,
  FormItem,
  FormLabel,
  FormMessage,
} from '@/components/ui/form'
import { Input } from '@/components/ui/input'

import type {
  BotProtectionFormValues,
  BotProtectionProvider,
  SecretConfigured,
} from './bot-protection-schema'

type BotProtectionProviderFieldsProps = {
  form: UseFormReturn<BotProtectionFormValues>
  provider: BotProtectionProvider
  secretConfigured: SecretConfigured
}

export function BotProtectionProviderFields(
  props: BotProtectionProviderFieldsProps
) {
  const { t } = useTranslation()
  const secretDescription = (configured: boolean) =>
    configured
      ? t('A secret key is configured. Leave blank to keep it.')
      : t('Required before enabling this provider')

  return (
    <div className='space-y-4 border-t pt-5'>
      <div>
        <h3 className='text-sm font-medium'>{t('Provider settings')}</h3>
        <p className='text-muted-foreground text-sm'>
          {t('Configure credentials for the selected provider')}
        </p>
      </div>

      {props.provider === 'turnstile' && (
        <>
          <FormField
            control={props.form.control}
            name='TurnstileSiteKey'
            render={({ field }) => (
              <FormItem>
                <FormLabel>{t('Site Key')}</FormLabel>
                <FormControl>
                  <Input
                    placeholder={t('Your Turnstile site key')}
                    autoComplete='off'
                    {...field}
                  />
                </FormControl>
                <FormMessage />
              </FormItem>
            )}
          />
          <FormField
            control={props.form.control}
            name='TurnstileSecretKey'
            render={({ field }) => (
              <FormItem>
                <FormLabel>{t('Secret Key')}</FormLabel>
                <FormControl>
                  <Input
                    type='password'
                    placeholder={t('Your Turnstile secret key')}
                    autoComplete='new-password'
                    {...field}
                  />
                </FormControl>
                <FormDescription>
                  {secretDescription(props.secretConfigured.turnstile)}
                </FormDescription>
                <FormMessage />
              </FormItem>
            )}
          />
        </>
      )}

      {props.provider === 'recaptcha' && (
        <>
          <FormField
            control={props.form.control}
            name='ReCaptchaSiteKey'
            render={({ field }) => (
              <FormItem>
                <FormLabel>{t('Site Key')}</FormLabel>
                <FormControl>
                  <Input
                    placeholder={t('Your reCAPTCHA site key')}
                    autoComplete='off'
                    {...field}
                  />
                </FormControl>
                <FormMessage />
              </FormItem>
            )}
          />
          <FormField
            control={props.form.control}
            name='ReCaptchaSecretKey'
            render={({ field }) => (
              <FormItem>
                <FormLabel>{t('Secret Key')}</FormLabel>
                <FormControl>
                  <Input
                    type='password'
                    placeholder={t('Your reCAPTCHA secret key')}
                    autoComplete='new-password'
                    {...field}
                  />
                </FormControl>
                <FormDescription>
                  {secretDescription(props.secretConfigured.recaptcha)}
                </FormDescription>
                <FormMessage />
              </FormItem>
            )}
          />
        </>
      )}

      {props.provider === 'geetest_v4' && (
        <>
          <FormField
            control={props.form.control}
            name='GeeTestCaptchaId'
            render={({ field }) => (
              <FormItem>
                <FormLabel>{t('Captcha ID')}</FormLabel>
                <FormControl>
                  <Input
                    placeholder={t('Your GeeTest V4 Captcha ID')}
                    autoComplete='off'
                    {...field}
                  />
                </FormControl>
                <FormMessage />
              </FormItem>
            )}
          />
          <FormField
            control={props.form.control}
            name='GeeTestSecretKey'
            render={({ field }) => (
              <FormItem>
                <FormLabel>{t('Captcha Key')}</FormLabel>
                <FormControl>
                  <Input
                    type='password'
                    placeholder={t('Your GeeTest V4 Captcha Key')}
                    autoComplete='new-password'
                    {...field}
                  />
                </FormControl>
                <FormDescription>
                  {secretDescription(props.secretConfigured.geetest_v4)}
                </FormDescription>
                <FormMessage />
              </FormItem>
            )}
          />
        </>
      )}

      {props.provider === 'cap' && (
        <>
          <FormField
            control={props.form.control}
            name='CapServerURL'
            render={({ field }) => (
              <FormItem>
                <FormLabel>{t('Server URL')}</FormLabel>
                <FormControl>
                  <Input
                    type='url'
                    placeholder='https://cap.example.com'
                    autoComplete='url'
                    {...field}
                  />
                </FormControl>
                <FormDescription>
                  {t('Public URL of your Cap Standalone instance')}
                </FormDescription>
                <FormMessage />
              </FormItem>
            )}
          />
          <FormField
            control={props.form.control}
            name='CapSiteKey'
            render={({ field }) => (
              <FormItem>
                <FormLabel>{t('Site Key')}</FormLabel>
                <FormControl>
                  <Input
                    placeholder={t('Your Cap site key')}
                    autoComplete='off'
                    {...field}
                  />
                </FormControl>
                <FormMessage />
              </FormItem>
            )}
          />
          <FormField
            control={props.form.control}
            name='CapSecretKey'
            render={({ field }) => (
              <FormItem>
                <FormLabel>{t('Secret Key')}</FormLabel>
                <FormControl>
                  <Input
                    type='password'
                    placeholder={t('Your Cap secret key')}
                    autoComplete='new-password'
                    {...field}
                  />
                </FormControl>
                <FormDescription>
                  {secretDescription(props.secretConfigured.cap)}
                </FormDescription>
                <FormMessage />
              </FormItem>
            )}
          />
        </>
      )}
    </div>
  )
}

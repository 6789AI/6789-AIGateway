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
import { zodResolver } from '@hookform/resolvers/zod'
import { ArrowDown, ArrowUp, Plus, Trash2 } from 'lucide-react'
import { useEffect, useMemo } from 'react'
import { useFieldArray, useForm } from 'react-hook-form'
import { useTranslation } from 'react-i18next'
import * as z from 'zod'

import { Button } from '@/components/ui/button'
import {
  Form,
  FormControl,
  FormDescription,
  FormField,
  FormItem,
  FormLabel,
  FormMessage,
} from '@/components/ui/form'
import { Input } from '@/components/ui/input'
import { Switch } from '@/components/ui/switch'
import {
  HEADER_NAV_LINK_KEYS,
  isSafeHeaderNavHref,
  MAX_HEADER_NAV_LINKS,
} from '@/lib/nav-modules'

import {
  SettingsControlChildren,
  SettingsForm,
  SettingsSwitchContent,
  SettingsControlGroup,
  SettingsSwitchItem,
} from '../components/settings-form-layout'
import { SettingsPageFormActions } from '../components/settings-page-context'
import { SettingsSection } from '../components/settings-section'
import { useUpdateOption } from '../hooks/use-update-option'
import {
  HEADER_NAV_DEFAULT,
  type HeaderNavModulesConfig,
  serializeHeaderNavModules,
} from './config'

const headerNavLinkSchema = z
  .object({
    key: z.string(),
    label: z.string().max(40, 'Maximum 40 characters.'),
    href: z.string().max(2048, 'Invalid navigation URL.'),
    enabled: z.boolean().optional(),
  })
  .superRefine((link, context) => {
    const isBuiltin = HEADER_NAV_LINK_KEYS.some((key) => key === link.key)
    if (!isBuiltin && !link.label.trim()) {
      context.addIssue({ code: 'custom', path: ['label'], message: 'Required' })
    }
    if (!isBuiltin && !link.href.trim()) {
      context.addIssue({ code: 'custom', path: ['href'], message: 'Required' })
    } else if (link.href.trim() && !isSafeHeaderNavHref(link.href.trim())) {
      context.addIssue({
        code: 'custom',
        path: ['href'],
        message: 'Invalid navigation URL.',
      })
    }
  })

const headerNavSchema = z.object({
  home: z.boolean(),
  console: z.boolean(),
  comfyCanvas: z.boolean(),
  pricingEnabled: z.boolean(),
  pricingRequireAuth: z.boolean(),
  rankingsEnabled: z.boolean(),
  rankingsRequireAuth: z.boolean(),
  docs: z.boolean(),
  about: z.boolean(),
  links: z.array(headerNavLinkSchema).max(MAX_HEADER_NAV_LINKS),
})

type HeaderNavFormValues = z.infer<typeof headerNavSchema>
type HeaderNavBooleanKey = Exclude<keyof HeaderNavFormValues, 'links'>

type HeaderNavigationSectionProps = {
  config: HeaderNavModulesConfig
  initialSerialized: string
}

const toFormValues = (config: HeaderNavModulesConfig): HeaderNavFormValues => ({
  home:
    config.home === undefined ? HEADER_NAV_DEFAULT.home : Boolean(config.home),
  console:
    config.console === undefined
      ? HEADER_NAV_DEFAULT.console
      : Boolean(config.console),
  comfyCanvas:
    config.comfyCanvas === undefined
      ? HEADER_NAV_DEFAULT.comfyCanvas
      : Boolean(config.comfyCanvas),
  pricingEnabled:
    config.pricing?.enabled === undefined
      ? HEADER_NAV_DEFAULT.pricing.enabled
      : Boolean(config.pricing.enabled),
  pricingRequireAuth:
    config.pricing?.requireAuth === undefined
      ? HEADER_NAV_DEFAULT.pricing.requireAuth
      : Boolean(config.pricing.requireAuth),
  rankingsEnabled:
    config.rankings?.enabled === undefined
      ? HEADER_NAV_DEFAULT.rankings.enabled
      : Boolean(config.rankings.enabled),
  rankingsRequireAuth:
    config.rankings?.requireAuth === undefined
      ? HEADER_NAV_DEFAULT.rankings.requireAuth
      : Boolean(config.rankings.requireAuth),
  docs:
    config.docs === undefined ? HEADER_NAV_DEFAULT.docs : Boolean(config.docs),
  about:
    config.about === undefined
      ? HEADER_NAV_DEFAULT.about
      : Boolean(config.about),
  links: config.links.map((link) => ({ ...link })),
})

export function HeaderNavigationSection({
  config,
  initialSerialized,
}: HeaderNavigationSectionProps) {
  const { t } = useTranslation()
  const updateOption = useUpdateOption()
  const formDefaults = useMemo(() => toFormValues(config), [config])

  const form = useForm<HeaderNavFormValues>({
    resolver: zodResolver(headerNavSchema),
    defaultValues: formDefaults,
  })
  const { fields, append, move, remove } = useFieldArray({
    control: form.control,
    name: 'links',
    keyName: 'fieldKey',
  })

  useEffect(() => {
    form.reset(formDefaults)
  }, [formDefaults, form])

  const onSubmit = async (values: HeaderNavFormValues) => {
    const payload: HeaderNavModulesConfig = {
      ...config,
      home: values.home,
      console: values.console,
      comfyCanvas: values.comfyCanvas,
      docs: values.docs,
      about: values.about,
      pricing: {
        ...(config.pricing ?? HEADER_NAV_DEFAULT.pricing),
        enabled: values.pricingEnabled,
        requireAuth: values.pricingRequireAuth,
      },
      rankings: {
        ...(config.rankings ?? HEADER_NAV_DEFAULT.rankings),
        enabled: values.rankingsEnabled,
        requireAuth: values.rankingsRequireAuth,
      },
      links: values.links.map((link) => ({
        ...link,
        label: link.label.trim(),
        href: link.href.trim(),
      })),
    }

    const serialized = serializeHeaderNavModules(payload)
    if (serialized === initialSerialized) {
      return
    }

    await updateOption.mutateAsync({
      key: 'HeaderNavModules',
      value: serialized,
    })
  }

  const resetToDefault = () => {
    form.reset(toFormValues(HEADER_NAV_DEFAULT))
  }

  const linkTitles: Record<string, string> = {
    home: t('Home'),
    console: t('Console'),
    comfyCanvas: t('Comfy Canvas'),
    pricing: t('Model Square'),
    rankings: t('Rankings'),
    docs: t('Docs'),
    about: t('About'),
  }
  const linkHrefs: Record<string, string> = {
    home: '/',
    console: '/dashboard',
    comfyCanvas: 'https://comfy.6789api.top/',
    pricing: '/pricing',
    rankings: '/rankings',
    docs: '/docs',
    about: '/about',
  }

  const simpleModules: Array<{
    key: HeaderNavBooleanKey
    title: string
    description: string
  }> = [
    {
      key: 'home',
      title: t('Home'),
      description: t('Landing page with system overview.'),
    },
    {
      key: 'console',
      title: t('Console'),
      description: t('User dashboard and quota controls.'),
    },
    {
      key: 'comfyCanvas',
      title: t('Comfy Canvas'),
      description: t('Open the Comfy Canvas website in a new tab.'),
    },
    {
      key: 'docs',
      title: t('Docs'),
      description: t('Documentation or external knowledge base.'),
    },
    {
      key: 'about',
      title: t('About'),
      description: t('Static page describing the platform.'),
    },
  ]

  const accessModules: Array<{
    enabledKey: HeaderNavBooleanKey
    requireAuthKey: HeaderNavBooleanKey
    requireAuthDependsOn: 'pricingEnabled' | 'rankingsEnabled'
    title: string
    description: string
    requireAuthTitle: string
    requireAuthDescription: string
  }> = [
    {
      enabledKey: 'pricingEnabled',
      requireAuthKey: 'pricingRequireAuth',
      requireAuthDependsOn: 'pricingEnabled',
      title: t('Model Square'),
      description: t('Public model catalog and pricing page.'),
      requireAuthTitle: t('Require login to view models'),
      requireAuthDescription: t(
        'Visitors must authenticate before accessing the pricing directory.'
      ),
    },
    {
      enabledKey: 'rankingsEnabled',
      requireAuthKey: 'rankingsRequireAuth',
      requireAuthDependsOn: 'rankingsEnabled',
      title: t('Rankings'),
      description: t('Public rankings page based on live usage data.'),
      requireAuthTitle: t('Require login to view rankings'),
      requireAuthDescription: t(
        'Visitors must authenticate before accessing the rankings page.'
      ),
    },
  ]

  return (
    <SettingsSection title={t('Header navigation')}>
      <Form {...form}>
        <SettingsForm onSubmit={form.handleSubmit(onSubmit)}>
          <SettingsPageFormActions
            onSave={form.handleSubmit(onSubmit)}
            onReset={resetToDefault}
            isSaving={updateOption.isPending}
            resetLabel='Reset to default'
            saveLabel='Save navigation'
          />
          <div className='grid gap-4 md:grid-cols-2'>
            {simpleModules.map((module) => (
              <FormField
                key={module.key}
                control={form.control}
                name={module.key}
                render={({ field }) => (
                  <SettingsSwitchItem>
                    <SettingsSwitchContent>
                      <FormLabel>{module.title}</FormLabel>
                      <FormDescription>{module.description}</FormDescription>
                    </SettingsSwitchContent>
                    <FormControl>
                      <Switch
                        checked={field.value}
                        onCheckedChange={field.onChange}
                      />
                    </FormControl>
                    <FormMessage />
                  </SettingsSwitchItem>
                )}
              />
            ))}
          </div>

          <div className='grid gap-4 lg:grid-cols-2'>
            {accessModules.map((module) => (
              <SettingsControlGroup key={module.enabledKey}>
                <FormField
                  control={form.control}
                  name={module.enabledKey}
                  render={({ field }) => (
                    <SettingsSwitchItem>
                      <SettingsSwitchContent>
                        <FormLabel>{module.title}</FormLabel>
                        <FormDescription>{module.description}</FormDescription>
                      </SettingsSwitchContent>
                      <FormControl>
                        <Switch
                          checked={field.value}
                          onCheckedChange={field.onChange}
                        />
                      </FormControl>
                      <FormMessage />
                    </SettingsSwitchItem>
                  )}
                />

                <FormField
                  control={form.control}
                  name={module.requireAuthKey}
                  render={({ field }) => (
                    <SettingsControlChildren>
                      <SettingsSwitchItem className='py-2'>
                        <SettingsSwitchContent>
                          <FormLabel>{module.requireAuthTitle}</FormLabel>
                          <FormDescription>
                            {module.requireAuthDescription}
                          </FormDescription>
                        </SettingsSwitchContent>
                        <FormControl>
                          <Switch
                            checked={field.value}
                            onCheckedChange={field.onChange}
                            disabled={!form.watch(module.requireAuthDependsOn)}
                          />
                        </FormControl>
                        <FormMessage />
                      </SettingsSwitchItem>
                    </SettingsControlChildren>
                  )}
                />
              </SettingsControlGroup>
            ))}
          </div>

          <div className='min-w-0 space-y-3'>
            <div className='flex items-center justify-between gap-3'>
              <h3 className='text-sm font-semibold'>{t('Navigation links')}</h3>
              <Button
                type='button'
                size='sm'
                variant='outline'
                disabled={fields.length >= MAX_HEADER_NAV_LINKS}
                onClick={() =>
                  append({
                    key: `custom-${crypto.randomUUID()}`,
                    label: '',
                    href: '',
                    enabled: true,
                  })
                }
              >
                <Plus data-icon='inline-start' />
                {t('Add link')}
              </Button>
            </div>

            <div className='divide-y rounded-lg border px-3'>
              {fields.map((link, index) => {
                const isBuiltin = HEADER_NAV_LINK_KEYS.some(
                  (key) => key === link.key
                )
                return (
                  <div key={link.fieldKey} className='min-w-0 space-y-3 py-3'>
                    <div className='flex justify-end'>
                      <div className='flex items-center gap-1'>
                        <Button
                          type='button'
                          size='icon-sm'
                          variant='ghost'
                          title={t('Move up')}
                          aria-label={t('Move up')}
                          disabled={index === 0}
                          onClick={() => move(index, index - 1)}
                        >
                          <ArrowUp />
                        </Button>
                        <Button
                          type='button'
                          size='icon-sm'
                          variant='ghost'
                          title={t('Move down')}
                          aria-label={t('Move down')}
                          disabled={index === fields.length - 1}
                          onClick={() => move(index, index + 1)}
                        >
                          <ArrowDown />
                        </Button>
                        {!isBuiltin && (
                          <Button
                            type='button'
                            size='icon-sm'
                            variant='ghost'
                            title={t('Remove')}
                            aria-label={t('Remove')}
                            onClick={() => remove(index)}
                          >
                            <Trash2 />
                          </Button>
                        )}
                      </div>
                    </div>

                    <div className='grid min-w-0 gap-3 sm:grid-cols-2'>
                      <FormField
                        control={form.control}
                        name={`links.${index}.label`}
                        render={({ field }) => (
                          <FormItem>
                            <FormLabel>{t('Button text')}</FormLabel>
                            <FormControl>
                              <Input
                                {...field}
                                placeholder={
                                  linkTitles[link.key] ?? t('Custom link')
                                }
                              />
                            </FormControl>
                            <FormMessage />
                          </FormItem>
                        )}
                      />
                      <FormField
                        control={form.control}
                        name={`links.${index}.href`}
                        render={({ field }) => (
                          <FormItem>
                            <FormLabel>{t('URL')}</FormLabel>
                            <FormControl>
                              <Input
                                {...field}
                                placeholder={
                                  linkHrefs[link.key] ?? 'https://example.com'
                                }
                              />
                            </FormControl>
                            {link.key === 'docs' && (
                              <FormDescription>
                                {t(
                                  'Leave blank to use the existing docs link.'
                                )}
                              </FormDescription>
                            )}
                            <FormMessage />
                          </FormItem>
                        )}
                      />
                    </div>
                    {!isBuiltin && (
                      <FormField
                        control={form.control}
                        name={`links.${index}.enabled`}
                        render={({ field }) => (
                          <FormItem className='flex items-center gap-2'>
                            <FormControl>
                              <Switch
                                checked={field.value !== false}
                                onCheckedChange={field.onChange}
                              />
                            </FormControl>
                            <FormLabel>{t('Visible')}</FormLabel>
                          </FormItem>
                        )}
                      />
                    )}
                  </div>
                )
              })}
            </div>
          </div>
        </SettingsForm>
      </Form>
    </SettingsSection>
  )
}

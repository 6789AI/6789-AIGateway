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
import { LayoutDashboard, Link2, Shield } from 'lucide-react'
import { useState } from 'react'
import { useTranslation } from 'react-i18next'

import { Dialog } from '@/components/dialog'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'

import type { UserProfile } from '../types'
import { LanguagePreferencesCard } from './language-preferences-card'
import { PasskeyCard } from './passkey-card'
import { ProfileSecurityCard } from './profile-security-card'
import { ProfileSettingsCard } from './profile-settings-card'
import { SidebarModulesCard } from './sidebar-modules-card'
import { TwoFACard } from './two-fa-card'

type ProfileSettingsDialogProps = {
  open: boolean
  onOpenChange: (open: boolean) => void
  profile: UserProfile | null
  loading: boolean
  onProfileUpdate: () => void
  canConfigureSidebar: boolean
}

export function ProfileSettingsDialog(props: ProfileSettingsDialogProps) {
  const { t } = useTranslation()
  const [activeTab, setActiveTab] = useState('account')

  const handleOpenChange = (open: boolean) => {
    if (!open) setActiveTab('account')
    props.onOpenChange(open)
  }

  return (
    <Dialog
      open={props.open}
      onOpenChange={handleOpenChange}
      title={t('Personal Settings')}
      description={t(
        'Manage account bindings, preferences, sidebar display, and security.'
      )}
      contentClassName='sm:max-w-5xl'
      contentHeight='min(78vh, 760px)'
      bodyClassName='space-y-4'
    >
      <Tabs value={activeTab} onValueChange={setActiveTab}>
        <TabsList className='grid w-full grid-cols-3 gap-1'>
          <TabsTrigger value='account' className='gap-2'>
            <Link2 className='size-4' />
            <span className='hidden sm:inline'>{t('Account & Notifications')}</span>
            <span className='sm:hidden'>{t('Account')}</span>
          </TabsTrigger>
          <TabsTrigger value='interface' className='gap-2'>
            <LayoutDashboard className='size-4' />
            <span className='hidden sm:inline'>{t('Interface & Sidebar')}</span>
            <span className='sm:hidden'>{t('Interface')}</span>
          </TabsTrigger>
          <TabsTrigger value='security' className='gap-2'>
            <Shield className='size-4' />
            <span className='hidden sm:inline'>{t('Security & Login')}</span>
            <span className='sm:hidden'>{t('Security')}</span>
          </TabsTrigger>
        </TabsList>

        <TabsContent value='account' className='mt-4'>
          <ProfileSettingsCard
            profile={props.profile}
            loading={props.loading}
            onProfileUpdate={props.onProfileUpdate}
          />
        </TabsContent>

        <TabsContent value='interface' className='mt-4 space-y-4 sm:space-y-6'>
          <LanguagePreferencesCard
            profile={props.profile}
            onProfileUpdate={props.onProfileUpdate}
          />
          {props.canConfigureSidebar && <SidebarModulesCard />}
        </TabsContent>

        <TabsContent value='security' className='mt-4 space-y-4 sm:space-y-6'>
          <ProfileSecurityCard profile={props.profile} loading={props.loading} />
          <PasskeyCard loading={props.loading} />
          <TwoFACard loading={props.loading} />
        </TabsContent>
      </Tabs>
    </Dialog>
  )
}

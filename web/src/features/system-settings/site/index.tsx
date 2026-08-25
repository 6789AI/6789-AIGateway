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
import { SettingsPage } from '../components/settings-page'
import type { SiteSettings } from '../types'
import {
  SITE_DEFAULT_SECTION,
  getSiteSectionContent,
  getSiteSectionMeta,
} from './section-registry.tsx'

const defaultSiteSettings: SiteSettings = {
  Notice: '',
  'global_banner.enabled': false,
  'global_banner.content': '',
  'global_banner.background_color': '#0EA5E9',
  'global_banner.text_color': '#082F49',
  'global_banner.icon': 'gift',
  'global_banner.countdown_enabled': false,
  'global_banner.countdown_end_at': 0,
  'global_banner.link_url': '/pricing',
  'global_banner.button_text': '',
  'global_banner.button_color': '#FFFFFF',
  'marketing_banner.enabled': false,
  'marketing_banner.content': '',
  'marketing_banner.background_color': '#A3E635',
  'marketing_banner.text_color': '#1A2E05',
  'marketing_banner.icon': 'megaphone',
  'marketing_banner.countdown_enabled': false,
  'marketing_banner.countdown_end_at': 0,
  'marketing_banner.link_url': '',
  'marketing_banner.button_text': '',
  'marketing_banner.button_color': '#FFFFFF',
  SystemName: 'New API',
  Logo: '',
  Footer: '',
  About: '',
  HomePageContent: '',
  ServerAddress: '',
  'legal.user_agreement': '',
  'legal.privacy_policy': '',
  HeaderNavModules: '',
  SidebarModulesAdmin: '',
}

export function SiteSettings() {
  return (
    <SettingsPage
      routePath='/_authenticated/system-settings/site/$section'
      defaultSettings={defaultSiteSettings}
      defaultSection={SITE_DEFAULT_SECTION}
      getSectionContent={getSiteSectionContent}
      getSectionMeta={getSiteSectionMeta}
    />
  )
}

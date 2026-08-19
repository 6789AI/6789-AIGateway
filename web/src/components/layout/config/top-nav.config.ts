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
import type { HeaderNavModules } from '../../../lib/nav-modules'
import type { TopNavLink } from '../types'

export const comfyCanvasTopNavLink = {
  href: 'https://comfy.6789api.top/',
  external: true,
} as const satisfies Pick<TopNavLink, 'href' | 'external'>

type BuildTopNavLinksOptions = {
  modules: HeaderNavModules
  docsLink?: string
  isAuthed: boolean
  translate: (key: string) => string
}

export function buildTopNavLinks(
  options: BuildTopNavLinksOptions
): TopNavLink[] {
  const links: TopNavLink[] = []

  if (options.modules.home !== false) {
    links.push({ title: options.translate('Home'), href: '/' })
  }

  if (options.modules.console !== false) {
    links.push({ title: options.translate('Console'), href: '/dashboard' })
  }

  if (options.modules.comfyCanvas !== false) {
    links.push({
      title: options.translate('Comfy Canvas'),
      ...comfyCanvasTopNavLink,
    })
  }

  if (options.modules.pricing.enabled) {
    links.push({
      title: options.translate('Model Square'),
      href: '/pricing',
      requiresAuth: options.modules.pricing.requireAuth && !options.isAuthed,
    })
  }

  if (options.modules.rankings.enabled) {
    links.push({
      title: options.translate('Rankings'),
      href: '/rankings',
      requiresAuth: options.modules.rankings.requireAuth && !options.isAuthed,
    })
  }

  if (options.modules.docs !== false) {
    links.push({
      title: options.translate('Docs'),
      href: options.docsLink ?? '/docs',
      external: Boolean(options.docsLink),
    })
  }

  if (options.modules.about !== false) {
    links.push({ title: options.translate('About'), href: '/about' })
  }

  return links
}

/**
 * Default top navigation links
 *
 * In practice, navigation links are dynamically fetched from backend.
 * Priority: Backend dynamic links > Provided navLinks > defaultTopNavLinks
 *
 * This is intentionally empty to encourage backend configuration.
 * If you need fallback links, add them here.
 */
export const defaultTopNavLinks: TopNavLink[] = []

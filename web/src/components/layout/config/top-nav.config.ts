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
import {
  isSafeHeaderNavHref,
  parseHeaderNavLinks,
  type HeaderNavModules,
} from '../../../lib/nav-modules'
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
  const docsHref =
    options.docsLink && isSafeHeaderNavHref(options.docsLink)
      ? options.docsLink
      : '/docs'
  const defaults = {
    home: { title: 'Home', href: '/', enabled: options.modules.home !== false },
    console: {
      title: 'Console',
      href: '/dashboard',
      enabled: options.modules.console !== false,
    },
    comfyCanvas: {
      title: 'Comfy Canvas',
      href: comfyCanvasTopNavLink.href,
      enabled: options.modules.comfyCanvas !== false,
    },
    pricing: {
      title: 'Model Square',
      href: '/pricing',
      enabled: options.modules.pricing.enabled,
    },
    rankings: {
      title: 'Rankings',
      href: '/rankings',
      enabled: options.modules.rankings.enabled,
    },
    docs: {
      title: 'Docs',
      href: docsHref,
      enabled: options.modules.docs !== false,
    },
    about: {
      title: 'About',
      href: '/about',
      enabled: options.modules.about !== false,
    },
  }

  return parseHeaderNavLinks(options.modules.links).flatMap((item) => {
    const builtin = defaults[item.key as keyof typeof defaults]
    if (builtin && !builtin.enabled) return []
    if (!builtin && item.enabled === false) return []

    const href = item.href || builtin?.href
    if (!href || !isSafeHeaderNavHref(href)) return []
    const title = item.label || (builtin && options.translate(builtin.title))
    if (!title) return []

    const isPricingRoute =
      href === '/pricing' ||
      href.startsWith('/pricing/') ||
      href.startsWith('/pricing?')
    const isRankingsRoute =
      href === '/rankings' ||
      href.startsWith('/rankings/') ||
      href.startsWith('/rankings?')

    return [
      {
        id: item.key,
        title,
        href,
        external: /^https?:\/\//i.test(href),
        requiresAuth:
          !options.isAuthed &&
          ((item.key === 'pricing' &&
            isPricingRoute &&
            options.modules.pricing.requireAuth) ||
            (item.key === 'rankings' &&
              isRankingsRoute &&
              options.modules.rankings.requireAuth)),
      },
    ]
  })
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

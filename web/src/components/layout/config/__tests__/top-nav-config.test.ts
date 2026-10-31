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
import { describe, test } from 'node:test'

import { buildTopNavLinks, comfyCanvasTopNavLink } from '../top-nav.config.ts'

const modules = {
  home: true,
  console: true,
  comfyCanvas: true,
  pricing: { enabled: true, requireAuth: false },
  rankings: { enabled: true, requireAuth: false },
  docs: true,
  about: true,
}

const translate = (key: string) => key

describe('top navigation configuration', () => {
  test('opens Comfy Canvas at the requested external URL', () => {
    assert.deepEqual(comfyCanvasTopNavLink, {
      href: 'https://comfy.6789api.top/',
      external: true,
    })
  })

  test('places Comfy Canvas before Model Square', () => {
    const links = buildTopNavLinks({ modules, isAuthed: false, translate })

    assert.deepEqual(
      links.map((link) => link.title),
      [
        'Home',
        'Console',
        'Comfy Canvas',
        'Model Square',
        'Rankings',
        'Docs',
        'About',
      ]
    )
  })

  test('hides Comfy Canvas when its navigation module is disabled', () => {
    const links = buildTopNavLinks({
      modules: { ...modules, comfyCanvas: false },
      isAuthed: false,
      translate,
    })

    assert.equal(
      links.some((link) => link.title === 'Comfy Canvas'),
      false
    )
  })

  test('uses saved order, labels, links and custom visibility', () => {
    const links = buildTopNavLinks({
      modules: {
        ...modules,
        links: [
          {
            key: 'custom-news',
            label: 'News',
            href: 'https://example.com/news',
          },
          { key: 'pricing', label: 'Models', href: '/pricing' },
          { key: 'home', label: '', href: '/welcome' },
          {
            key: 'custom-hidden',
            label: 'Hidden',
            href: '/hidden',
            enabled: false,
          },
        ],
      },
      isAuthed: false,
      translate: (key) => `translated:${key}`,
    })

    assert.deepEqual(
      links.slice(0, 3).map(({ title, href, external }) => ({
        title,
        href,
        external,
      })),
      [
        { title: 'News', href: 'https://example.com/news', external: true },
        { title: 'Models', href: '/pricing', external: false },
        { title: 'translated:Home', href: '/welcome', external: false },
      ]
    )
    assert.equal(
      links.some((link) => link.title === 'Hidden'),
      false
    )
    assert.equal(links.length, 8)
  })

  test('keeps route access behavior for the original pricing route', () => {
    const links = buildTopNavLinks({
      modules: {
        ...modules,
        pricing: { enabled: true, requireAuth: true },
        links: [
          { key: 'pricing', label: '', href: '/pricing' },
          { key: 'rankings', label: '', href: '/welcome' },
        ],
      },
      isAuthed: false,
      translate,
    })

    assert.equal(
      links.find((link) => link.id === 'pricing')?.requiresAuth,
      true
    )
    assert.equal(
      links.find((link) => link.id === 'rankings')?.requiresAuth,
      false
    )
  })
})

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
import { useId } from 'react'
import { useTranslation } from 'react-i18next'

import {
  Field,
  FieldDescription,
  FieldError,
  FieldLabel,
} from '@/components/ui/field'
import {
  InputGroup,
  InputGroupAddon,
  InputGroupInput,
} from '@/components/ui/input-group'

import { VIDEO_RESOLUTIONS } from './video-resolution-prices'

type Props = {
  prices: Record<string, string>
  onChange: (prices: Record<string, string>) => void
  multiplyByDuration: boolean
  invalid: boolean
}

export function VideoResolutionPriceEditor(props: Props) {
  const { t } = useTranslation()
  const id = useId()
  return (
    <fieldset className='grid gap-3' aria-describedby={`${id}-description`}>
      <legend className='mb-2 text-sm font-medium'>
        {t('Video quality prices')}
      </legend>
      <FieldDescription id={`${id}-description`}>
        {t(
          'Leave a quality price blank to use the base price. A configured quality price replaces the base price and built-in resolution ratios.'
        )}
      </FieldDescription>
      <div className='grid gap-3 sm:grid-cols-2'>
        {VIDEO_RESOLUTIONS.map((resolution) => (
          <Field key={resolution}>
            <FieldLabel htmlFor={`${id}-${resolution}`}>
              {resolution.toUpperCase().replace('P', 'p')}
            </FieldLabel>
            <InputGroup>
              <InputGroupAddon>$</InputGroupAddon>
              <InputGroupInput
                id={`${id}-${resolution}`}
                inputMode='decimal'
                placeholder={t('Use base price')}
                value={props.prices[resolution] || ''}
                aria-invalid={props.invalid || undefined}
                aria-describedby={props.invalid ? `${id}-error` : undefined}
                onChange={(event) =>
                  props.onChange({
                    ...props.prices,
                    [resolution]: event.target.value,
                  })
                }
              />
              <InputGroupAddon align='inline-end'>
                {props.multiplyByDuration ? t('per second') : t('per request')}
              </InputGroupAddon>
            </InputGroup>
          </Field>
        ))}
      </div>
      {props.invalid && (
        <FieldError id={`${id}-error`}>
          {t(
            'Video quality prices must be finite numbers greater than or equal to zero.'
          )}
        </FieldError>
      )}
    </fieldset>
  )
}

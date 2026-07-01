/*
Copyright (C) 2023-2026 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.
*/

import { useTranslation } from 'react-i18next'
import {
  Accordion,
  AccordionContent,
  AccordionItem,
  AccordionTrigger,
} from '@/components/ui/accordion'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'

export function SeedanceUsageGuide() {
  const { t } = useTranslation()

  return (
    <Card className='mb-4'>
      <CardHeader className='pb-2'>
        <CardTitle className='text-base'>
          {t('How to use Seedance assets in video generation')}
        </CardTitle>
      </CardHeader>
      <CardContent className='space-y-3 text-sm'>
        <ol className='text-muted-foreground list-decimal space-y-1 pl-5'>
          <li>{t('Upload assets here and copy the seedance_asset:// reference.')}</li>
          <li>{t('Call POST /v1/video/generations with your API key (same account).')}</li>
          <li>{t('Poll GET /v1/video/generations/{task_id} until status is completed.')}</li>
        </ol>
        <Accordion className='w-full'>
          <AccordionItem value='jd20'>
            <AccordionTrigger className='text-sm'>
              {t('JDseedance2.0-10 (dance-create) example')}
            </AccordionTrigger>
            <AccordionContent>
              <pre className='bg-muted overflow-x-auto rounded-md p-3 text-xs leading-relaxed'>
                {`curl https://YOUR_DOMAIN/v1/video/generations \\
  -H "Authorization: Bearer sk-YOUR_KEY" \\
  -H "Content-Type: application/json" \\
  -d '{
    "model": "JDseedance2.0-10",
    "prompt": "让参考图动起来",
    "duration": 11,
    "metadata": {
      "ratio": "16:9",
      "generate_audio": true,
      "watermark": false,
      "content": [{
        "type": "image_url",
        "image_url": { "url": "seedance_asset://1" },
        "role": "reference_image"
      }]
    }
  }'`}
              </pre>
            </AccordionContent>
          </AccordionItem>
          <AccordionItem value='legacy'>
            <AccordionTrigger className='text-sm'>
              {t('JDSeedance legacy (type 58) example')}
            </AccordionTrigger>
            <AccordionContent>
              <pre className='bg-muted overflow-x-auto rounded-md p-3 text-xs leading-relaxed'>
                {`curl https://YOUR_DOMAIN/v1/video/generations \\
  -H "Authorization: Bearer sk-YOUR_KEY" \\
  -H "Content-Type: application/json" \\
  -d '{
    "model": "drrfsmvr2.0",
    "prompt": "让参考图动起来",
    "duration": 5,
    "size": "480p",
    "images": ["seedance_asset://1"]
  }'`}
              </pre>
            </AccordionContent>
          </AccordionItem>
        </Accordion>
        <p className='text-muted-foreground text-xs'>
          {t(
            'Use only one sk- prefix in Authorization. The asset ID must belong to the same user as the API key.',
          )}
        </p>
      </CardContent>
    </Card>
  )
}

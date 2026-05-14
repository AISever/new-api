/*
Copyright (C) 2025 QuantumNous

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

import React from 'react';
import { Typography } from '@douyinfe/semi-ui';
import {
  Moonshot,
  OpenAI,
  XAI,
  Zhipu,
  Volcengine,
  Cohere,
  Claude,
  Gemini,
  Suno,
  Minimax,
  Wenxin,
  Spark,
  Qingyan,
  DeepSeek,
  Qwen,
  Midjourney,
  Grok,
  AzureAI,
  Hunyuan,
  Xinference,
} from '@lobehub/icons';

const providers = [
  { key: 'moonshot', icon: <Moonshot size={40} /> },
  { key: 'openai', icon: <OpenAI size={40} /> },
  { key: 'xai', icon: <XAI size={40} /> },
  { key: 'zhipu', icon: <Zhipu.Color size={40} /> },
  { key: 'volcengine', icon: <Volcengine.Color size={40} /> },
  { key: 'cohere', icon: <Cohere.Color size={40} /> },
  { key: 'claude', icon: <Claude.Color size={40} /> },
  { key: 'gemini', icon: <Gemini.Color size={40} /> },
  { key: 'suno', icon: <Suno size={40} /> },
  { key: 'minimax', icon: <Minimax.Color size={40} /> },
  { key: 'wenxin', icon: <Wenxin.Color size={40} /> },
  { key: 'spark', icon: <Spark.Color size={40} /> },
  { key: 'qingyan', icon: <Qingyan.Color size={40} /> },
  { key: 'deepseek', icon: <DeepSeek.Color size={40} /> },
  { key: 'qwen', icon: <Qwen.Color size={40} /> },
  { key: 'midjourney', icon: <Midjourney size={40} /> },
  { key: 'grok', icon: <Grok size={40} /> },
  { key: 'azureai', icon: <AzureAI.Color size={40} /> },
  { key: 'hunyuan', icon: <Hunyuan.Color size={40} /> },
  { key: 'xinference', icon: <Xinference.Color size={40} /> },
];

const ProviderLogoCloud = ({ title }) => {
  return (
    <div className='mt-12 md:mt-16 lg:mt-20 w-full'>
      <div className='flex items-center mb-6 md:mb-8 justify-center'>
        <Typography.Text
          type='tertiary'
          className='text-lg md:text-xl lg:text-2xl font-light'
        >
          {title}
        </Typography.Text>
      </div>
      <div className='flex flex-wrap items-center justify-center gap-3 sm:gap-4 md:gap-6 lg:gap-8 max-w-5xl mx-auto px-4'>
        {providers.map((provider) => (
          <div
            key={provider.key}
            className='w-8 h-8 sm:w-10 sm:h-10 md:w-12 md:h-12 flex items-center justify-center'
          >
            {provider.icon}
          </div>
        ))}
        <div className='w-8 h-8 sm:w-10 sm:h-10 md:w-12 md:h-12 flex items-center justify-center'>
          <Typography.Text className='!text-lg sm:!text-xl md:!text-2xl lg:!text-3xl font-bold'>
            30+
          </Typography.Text>
        </div>
      </div>
    </div>
  );
};

export default ProviderLogoCloud;

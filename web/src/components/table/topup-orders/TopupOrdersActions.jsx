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
import { Banner, Button } from '@douyinfe/semi-ui';
import { IconRefresh } from '@douyinfe/semi-icons';

const TopupOrdersActions = ({ refresh, loading, t }) => {
  return (
    <div className='flex flex-col md:flex-row justify-between items-start md:items-center gap-2 w-full'>
      <Button
        icon={<IconRefresh />}
        onClick={refresh}
        loading={loading}
        type='tertiary'
        size='small'
      >
        {t('刷新')}
      </Button>

      <Banner
        type='info'
        closeIcon={null}
        className='!rounded-lg'
        description={t('待支付的在线支付订单会在列表加载时自动同步，无需用户补单。')}
      />
    </div>
  );
};

export default TopupOrdersActions;


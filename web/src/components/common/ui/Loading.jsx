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

const Loading = ({ size = 'small' }) => {
  const spinnerSize = size === 'large' ? 'h-10 w-10' : 'h-7 w-7';

  return (
    <div className='fixed inset-0 w-screen h-screen flex items-center justify-center'>
      <div
        className={`${spinnerSize} rounded-full border-2 border-semi-color-border border-t-semi-color-primary animate-spin`}
      />
    </div>
  );
};

export default Loading;

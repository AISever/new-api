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

import React, { Suspense, lazy } from 'react';
import { Route, Routes, useLocation } from 'react-router-dom';
import { AdminRoute, PrivateRoute } from '../../helpers';
import Loading from '../common/ui/Loading';
import { lazyWithRetry } from '../../helpers/lazyWithRetry';

const Dashboard = lazy(lazyWithRetry(() => import('../../pages/Dashboard'), 'dashboard'));
const User = lazy(lazyWithRetry(() => import('../../pages/User'), 'user'));
const Setting = lazy(lazyWithRetry(() => import('../../pages/Setting'), 'setting'));
const Channel = lazy(lazyWithRetry(() => import('../../pages/Channel'), 'channel'));
const Token = lazy(lazyWithRetry(() => import('../../pages/Token'), 'token'));
const Redemption = lazy(lazyWithRetry(() => import('../../pages/Redemption'), 'redemption'));
const TopUp = lazy(lazyWithRetry(() => import('../../pages/TopUp'), 'topup'));
const Log = lazy(lazyWithRetry(() => import('../../pages/Log'), 'log'));
const Chat = lazy(lazyWithRetry(() => import('../../pages/Chat'), 'chat'));
const Midjourney = lazy(lazyWithRetry(() => import('../../pages/Midjourney'), 'midjourney'));
const Task = lazy(lazyWithRetry(() => import('../../pages/Task'), 'task'));
const ModelPage = lazy(lazyWithRetry(() => import('../../pages/Model'), 'models'));
const ModelDeploymentPage = lazy(
  lazyWithRetry(() => import('../../pages/ModelDeployment'), 'deployment'),
);
const Playground = lazy(lazyWithRetry(() => import('../../pages/Playground'), 'playground'));
const Subscription = lazy(
  lazyWithRetry(() => import('../../pages/Subscription'), 'subscription'),
);
const PersonalSetting = lazy(
  lazyWithRetry(() => import('../settings/PersonalSetting'), 'personal-setting'),
);
const NotFound = lazy(lazyWithRetry(() => import('../../pages/NotFound'), 'console-not-found'));

const ConsoleRoutes = () => {
  const location = useLocation();

  const wrap = (element) => (
    <Suspense fallback={<Loading></Loading>} key={location.pathname}>
      {element}
    </Suspense>
  );

  return (
    <Routes>
      <Route
        path='/'
        element={<PrivateRoute>{wrap(<Dashboard />)}</PrivateRoute>}
      />
      <Route
        path='models'
        element={<AdminRoute>{wrap(<ModelPage />)}</AdminRoute>}
      />
      <Route
        path='deployment'
        element={<AdminRoute>{wrap(<ModelDeploymentPage />)}</AdminRoute>}
      />
      <Route
        path='subscription'
        element={<AdminRoute>{wrap(<Subscription />)}</AdminRoute>}
      />
      <Route
        path='channel'
        element={<AdminRoute>{wrap(<Channel />)}</AdminRoute>}
      />
      <Route
        path='token'
        element={<PrivateRoute>{wrap(<Token />)}</PrivateRoute>}
      />
      <Route
        path='playground'
        element={<PrivateRoute>{wrap(<Playground />)}</PrivateRoute>}
      />
      <Route
        path='redemption'
        element={<AdminRoute>{wrap(<Redemption />)}</AdminRoute>}
      />
      <Route
        path='user'
        element={<AdminRoute>{wrap(<User />)}</AdminRoute>}
      />
      <Route
        path='setting'
        element={<AdminRoute>{wrap(<Setting />)}</AdminRoute>}
      />
      <Route
        path='personal'
        element={<PrivateRoute>{wrap(<PersonalSetting />)}</PrivateRoute>}
      />
      <Route
        path='topup'
        element={<PrivateRoute>{wrap(<TopUp />)}</PrivateRoute>}
      />
      <Route
        path='log'
        element={<PrivateRoute>{wrap(<Log />)}</PrivateRoute>}
      />
      <Route
        path='midjourney'
        element={<PrivateRoute>{wrap(<Midjourney />)}</PrivateRoute>}
      />
      <Route
        path='task'
        element={<PrivateRoute>{wrap(<Task />)}</PrivateRoute>}
      />
      <Route
        path='chat/:id?'
        element={<PrivateRoute>{wrap(<Chat />)}</PrivateRoute>}
      />
      <Route path='*' element={wrap(<NotFound />)} />
    </Routes>
  );
};

export default ConsoleRoutes;

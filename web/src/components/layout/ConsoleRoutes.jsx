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

const Dashboard = lazy(() => import('../../pages/Dashboard'));
const User = lazy(() => import('../../pages/User'));
const Setting = lazy(() => import('../../pages/Setting'));
const Channel = lazy(() => import('../../pages/Channel'));
const Token = lazy(() => import('../../pages/Token'));
const Redemption = lazy(() => import('../../pages/Redemption'));
const TopUp = lazy(() => import('../../pages/TopUp'));
const Log = lazy(() => import('../../pages/Log'));
const Chat = lazy(() => import('../../pages/Chat'));
const Midjourney = lazy(() => import('../../pages/Midjourney'));
const Task = lazy(() => import('../../pages/Task'));
const ModelPage = lazy(() => import('../../pages/Model'));
const ModelDeploymentPage = lazy(() => import('../../pages/ModelDeployment'));
const Playground = lazy(() => import('../../pages/Playground'));
const Subscription = lazy(() => import('../../pages/Subscription'));
const PersonalSetting = lazy(() => import('../settings/PersonalSetting'));
const NotFound = lazy(() => import('../../pages/NotFound'));

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

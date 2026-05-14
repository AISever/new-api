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

import React, { lazy, Suspense, useContext, useMemo } from 'react';
import { Route, Routes, useLocation, useParams } from 'react-router-dom';
import Loading from './components/common/ui/Loading';
import { AuthRedirect, PrivateRoute, AdminRoute } from './helpers/auth';
import { StatusContext } from './context/Status';
import OAuth2Callback from './components/auth/OAuth2Callback';
import SetupCheck from './components/layout/SetupCheck';
const PageLayout = lazy(() => import('./components/layout/PageLayout'));
const PublicLayout = lazy(() => import('./components/layout/PublicLayout'));

const Home = lazy(() => import('./pages/Home'));
const Dashboard = lazy(() => import('./pages/Dashboard'));
const About = lazy(() => import('./pages/About'));
const RegisterForm = lazy(() => import('./components/auth/RegisterForm'));
const LoginForm = lazy(() => import('./components/auth/LoginForm'));
const PasswordResetForm = lazy(() => import('./components/auth/PasswordResetForm'));
const PasswordResetConfirm = lazy(() =>
  import('./components/auth/PasswordResetConfirm'),
);
const NotFound = lazy(() => import('./pages/NotFound'));
const Forbidden = lazy(() => import('./pages/Forbidden'));
const User = lazy(() => import('./pages/User'));
const Setting = lazy(() => import('./pages/Setting'));
const Channel = lazy(() => import('./pages/Channel'));
const Token = lazy(() => import('./pages/Token'));
const Redemption = lazy(() => import('./pages/Redemption'));
const TopUp = lazy(() => import('./pages/TopUp'));
const Log = lazy(() => import('./pages/Log'));
const Chat = lazy(() => import('./pages/Chat'));
const Chat2Link = lazy(() => import('./pages/Chat2Link'));
const Midjourney = lazy(() => import('./pages/Midjourney'));
const Pricing = lazy(() => import('./pages/Pricing'));
const Task = lazy(() => import('./pages/Task'));
const ModelPage = lazy(() => import('./pages/Model'));
const ModelDeploymentPage = lazy(() => import('./pages/ModelDeployment'));
const Playground = lazy(() => import('./pages/Playground'));
const Subscription = lazy(() => import('./pages/Subscription'));
const PersonalSetting = lazy(() => import('./components/settings/PersonalSetting'));
const Setup = lazy(() => import('./pages/Setup'));
const UserAgreement = lazy(() => import('./pages/UserAgreement'));
const PrivacyPolicy = lazy(() => import('./pages/PrivacyPolicy'));
const Help = lazy(() => import('./pages/Help'));
const Docs = lazy(() => import('./pages/Docs'));

function DynamicOAuth2Callback() {
  const { provider } = useParams();
  return <OAuth2Callback type={provider} />;
}

function App() {
  const location = useLocation();
  const [statusState] = useContext(StatusContext);

  // 获取模型广场权限配置
  const pricingRequireAuth = useMemo(() => {
    const headerNavModulesConfig = statusState?.status?.HeaderNavModules;
    if (headerNavModulesConfig) {
      try {
        const modules = JSON.parse(headerNavModulesConfig);

        // 处理向后兼容性：如果pricing是boolean，默认不需要登录
        if (typeof modules.pricing === 'boolean') {
          return false; // 默认不需要登录鉴权
        }

        // 如果是对象格式，使用requireAuth配置
        return modules.pricing?.requireAuth === true;
      } catch (error) {
        console.error('解析顶栏模块配置失败:', error);
        return false; // 默认不需要登录
      }
    }
    return false; // 默认不需要登录
  }, [statusState?.status?.HeaderNavModules]);

  return (
    <SetupCheck>
      <Routes>
        <Route
          element={
            <Suspense fallback={<Loading></Loading>} key={`public-${location.pathname}`}>
              <PublicLayout />
            </Suspense>
          }
        >
          <Route
            path='/'
            element={
              <Suspense fallback={<Loading></Loading>} key={location.pathname}>
                <Home />
              </Suspense>
            }
          />
          <Route
            path='/pricing'
            element={
              pricingRequireAuth ? (
                <PrivateRoute>
                  <Suspense
                    fallback={<Loading></Loading>}
                    key={location.pathname}
                  >
                    <Pricing />
                  </Suspense>
                </PrivateRoute>
              ) : (
                <Suspense fallback={<Loading></Loading>} key={location.pathname}>
                  <Pricing />
                </Suspense>
              )
            }
          />
          <Route
            path='/about'
            element={
              <Suspense fallback={<Loading></Loading>} key={location.pathname}>
                <About />
              </Suspense>
            }
          />
          <Route
            path='/help'
            element={
              <Suspense fallback={<Loading></Loading>} key={location.pathname}>
                <Help />
              </Suspense>
            }
          />
          <Route
            path='/docs/:docId?'
            element={
              <Suspense fallback={<Loading></Loading>} key={location.pathname}>
                <Docs />
              </Suspense>
            }
          />
        </Route>
        <Route
          path='/console/*'
          element={
            <Suspense fallback={<Loading></Loading>} key={location.pathname}>
              <PageLayout />
            </Suspense>
          }
        />
        <Route
          path='/setup'
          element={
            <Suspense fallback={<Loading></Loading>} key={location.pathname}>
              <Setup />
            </Suspense>
          }
        />
        <Route path='/forbidden' element={<Forbidden />} />
        <Route
          path='/user/reset'
          element={
            <Suspense fallback={<Loading></Loading>} key={location.pathname}>
              <PasswordResetConfirm />
            </Suspense>
          }
        />
        <Route
          path='/login'
          element={
            <Suspense fallback={<Loading></Loading>} key={location.pathname}>
              <AuthRedirect>
                <LoginForm />
              </AuthRedirect>
            </Suspense>
          }
        />
        <Route
          path='/register'
          element={
            <Suspense fallback={<Loading></Loading>} key={location.pathname}>
              <AuthRedirect>
                <RegisterForm />
              </AuthRedirect>
            </Suspense>
          }
        />
        <Route
          path='/reset'
          element={
            <Suspense fallback={<Loading></Loading>} key={location.pathname}>
              <PasswordResetForm />
            </Suspense>
          }
        />
        <Route
          path='/oauth/github'
          element={
            <Suspense fallback={<Loading></Loading>} key={location.pathname}>
              <OAuth2Callback type='github'></OAuth2Callback>
            </Suspense>
          }
        />
        <Route
          path='/oauth/discord'
          element={
            <Suspense fallback={<Loading></Loading>} key={location.pathname}>
              <OAuth2Callback type='discord'></OAuth2Callback>
            </Suspense>
          }
        />
        <Route
          path='/oauth/oidc'
          element={
            <Suspense fallback={<Loading></Loading>}>
              <OAuth2Callback type='oidc'></OAuth2Callback>
            </Suspense>
          }
        />
        <Route
          path='/oauth/linuxdo'
          element={
            <Suspense fallback={<Loading></Loading>} key={location.pathname}>
              <OAuth2Callback type='linuxdo'></OAuth2Callback>
            </Suspense>
          }
        />
        <Route
          path='/oauth/:provider'
          element={
            <Suspense fallback={<Loading></Loading>} key={location.pathname}>
              <DynamicOAuth2Callback />
            </Suspense>
          }
        />
        <Route
          path='/user-agreement'
          element={
            <Suspense fallback={<Loading></Loading>} key={location.pathname}>
              <UserAgreement />
            </Suspense>
          }
        />
        <Route
          path='/privacy-policy'
          element={
            <Suspense fallback={<Loading></Loading>} key={location.pathname}>
              <PrivacyPolicy />
            </Suspense>
          }
        />
        <Route path='*' element={<NotFound />} />
      </Routes>
    </SetupCheck>
  );
}

export default App;

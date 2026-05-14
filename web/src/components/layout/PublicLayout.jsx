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

import '@douyinfe/semi-ui/dist/css/semi.css';
import React, { useMemo } from 'react';
import { Layout, LocaleProvider } from '@douyinfe/semi-ui';
import { Outlet, useLocation } from 'react-router-dom';
import { useTranslation } from 'react-i18next';
import zh_CN from '@douyinfe/semi-ui/lib/es/locale/source/zh_CN';
import en_GB from '@douyinfe/semi-ui/lib/es/locale/source/en_GB';
import HeaderBar from './headerbar';
import FooterBar from './Footer';
import ErrorBoundary from '../common/ErrorBoundary';
import { shouldHidePublicFooter } from './routeShells';

const { Header, Content, Footer } = Layout;

const PublicLayout = () => {
  const location = useLocation();
  const { i18n } = useTranslation();
  const semiLocale = useMemo(
    () => ({ zh: zh_CN, en: en_GB })[i18n.language] || zh_CN,
    [i18n.language],
  );
  const hideFooter = shouldHidePublicFooter(location.pathname);

  return (
    <LocaleProvider locale={semiLocale}>
      <Layout
        className='app-layout'
        style={{
          display: 'flex',
          flexDirection: 'column',
          minHeight: '100vh',
        }}
      >
        <Header
          style={{
            padding: 0,
            height: 'auto',
            lineHeight: 'normal',
            position: 'fixed',
            width: '100%',
            top: 0,
            zIndex: 100,
          }}
        >
          <HeaderBar drawerOpen={false} />
        </Header>
        <Content
          style={{
            flex: '1 1 auto',
            paddingTop: '64px',
            position: 'relative',
          }}
        >
          <ErrorBoundary resetKey={location.pathname}>
            <Outlet />
          </ErrorBoundary>
        </Content>
        {!hideFooter && (
          <Footer
            style={{
              flex: '0 0 auto',
              width: '100%',
            }}
          >
            <FooterBar />
          </Footer>
        )}
      </Layout>
    </LocaleProvider>
  );
};

export default PublicLayout;

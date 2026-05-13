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

import { useContext, useEffect } from 'react';
import { toast } from 'react-toastify';
import { API } from '../../helpers/api';
import { applyBrandingToDocument } from '../../helpers/branding';
import { setStatusData } from '../../helpers/data';
import { getLogo, getSystemName } from '../../helpers/utils';
import { UserContext } from '../../context/User';
import { StatusContext } from '../../context/Status';
import { normalizeLanguage } from '../../i18n/language';
import { useTranslation } from 'react-i18next';

export const useAppBootstrap = () => {
  const [userState, userDispatch] = useContext(UserContext);
  const [, statusDispatch] = useContext(StatusContext);
  const { i18n } = useTranslation();

  useEffect(() => {
    const loadUser = () => {
      const raw = localStorage.getItem('user');
      if (!raw) {
        return;
      }

      try {
        userDispatch({ type: 'login', payload: JSON.parse(raw) });
      } catch (error) {
        console.error('Failed to parse user from localStorage', error);
      }
    };

    const loadStatus = async () => {
      try {
        const res = await API.get('/api/status');
        const { success, data } = res.data;
        if (!success) {
          toast.error('Unable to connect to server');
          return;
        }

        statusDispatch({ type: 'set', payload: data });
        setStatusData(data);
        applyBrandingToDocument(document, {
          systemName: data.system_name,
          logo: data.logo,
        });
      } catch (error) {
        console.error('Failed to load status', error);
        toast.error('Failed to load status');
      }
    };

    loadUser();
    loadStatus().catch(console.error);
    applyBrandingToDocument(document, {
      systemName: getSystemName(),
      logo: getLogo(),
    });
  }, [statusDispatch, userDispatch]);

  useEffect(() => {
    let preferredLang;

    if (userState?.user?.setting) {
      try {
        const settings = JSON.parse(userState.user.setting);
        preferredLang = normalizeLanguage(settings.language);
      } catch (error) {
        console.error('Failed to parse language settings', error);
      }
    }

    if (!preferredLang) {
      const savedLang = localStorage.getItem('i18nextLng');
      if (savedLang) {
        preferredLang = normalizeLanguage(savedLang);
      }
    }

    if (preferredLang) {
      localStorage.setItem('i18nextLng', preferredLang);
      if (preferredLang !== i18n.language) {
        i18n.changeLanguage(preferredLang);
      }
    }
  }, [i18n, userState?.user?.setting]);
};

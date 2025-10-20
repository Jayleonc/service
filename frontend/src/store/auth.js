import { defineStore } from 'pinia';
import { ref, computed } from 'vue';

import { login as apiLogin, register as apiRegister, fetchProfile } from '@/api/auth';
import { setAuthToken } from '@/api/client';

const ACCESS_TOKEN_KEY = 'service_access_token';
const REFRESH_TOKEN_KEY = 'service_refresh_token';

export const useAuthStore = defineStore('auth', () => {
  const token = ref(localStorage.getItem(ACCESS_TOKEN_KEY) || '');
  const refreshToken = ref(localStorage.getItem(REFRESH_TOKEN_KEY) || '');
  const user = ref(null);

  if (token.value) {
    setAuthToken(token.value);
  }

  const roles = computed(() => user.value?.roles || []);
  const isAuthenticated = computed(() => !!token.value);
  const isAdmin = computed(() => roles.value.includes('ADMIN'));

  const setSession = (session) => {
    token.value = session.access_token || '';
    refreshToken.value = session.refresh_token || '';
    user.value = session.user || null;

    setAuthToken(token.value);
    if (token.value) {
      localStorage.setItem(ACCESS_TOKEN_KEY, token.value);
    } else {
      localStorage.removeItem(ACCESS_TOKEN_KEY);
    }

    if (refreshToken.value) {
      localStorage.setItem(REFRESH_TOKEN_KEY, refreshToken.value);
    } else {
      localStorage.removeItem(REFRESH_TOKEN_KEY);
    }
  };

  const login = async (credentials) => {
    const data = await apiLogin(credentials);
    setSession({
      access_token: data.access_token,
      refresh_token: data.refresh_token,
      user: data.user,
    });
    return data.user;
  };

  const register = async (userData) => {
    const data = await apiRegister(userData);
    return data;
  };

  const loadProfile = async () => {
    if (!token.value) {
      return;
    }
    const profile = await fetchProfile();
    user.value = profile;
  };

  const logout = () => {
    setAuthToken('');
    token.value = '';
    refreshToken.value = '';
    user.value = null;
    localStorage.removeItem(ACCESS_TOKEN_KEY);
    localStorage.removeItem(REFRESH_TOKEN_KEY);
  };

  return {
    token,
    refreshToken,
    user,
    roles,
    isAuthenticated,
    isAdmin,
    login,
    register,
    logout,
    loadProfile,
    setSession,
  };
});

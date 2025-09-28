import client from './client';

export const login = async (payload) => {
  const { data } = await client.post('/users/login', payload);
  return data.data;
};

export const register = async (payload) => {
  const { data } = await client.post('/users/register', payload);
  return data.data;
};

export const fetchProfile = async () => {
  const { data } = await client.post('/users/me/get', {});
  return data.data;
};

export const updateProfile = async (payload) => {
  const { data } = await client.post('/users/me/update', payload);
  return data.data;
};

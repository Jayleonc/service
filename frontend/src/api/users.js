import client from './client';

export const listUsers = async (pagination, filters = {}) => {
  const payload = {
    pagination,
    ...filters,
  };
  const { data } = await client.post('/user/list', payload);
  return data.data;
};

export const createUser = async (payload) => {
  const { data } = await client.post('/user/create', payload);
  return data.data;
};

export const updateUser = async (payload) => {
  const { data } = await client.post('/user/update', payload);
  return data.data;
};

export const deleteUser = async (id) => {
  const { data } = await client.post('/user/delete', { id });
  return data.data;
};

export const assignUserRoles = async (payload) => {
  const { data } = await client.post('/user/assign_roles', payload);
  return data.data;
};

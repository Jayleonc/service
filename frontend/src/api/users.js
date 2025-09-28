import client from './client';

export const listUsers = async (pagination, filters = {}) => {
  const payload = {
    pagination,
    ...filters,
  };
  const { data } = await client.post('/users/list', payload);
  return data.data;
};

export const createUser = async (payload) => {
  const { data } = await client.post('/users/create', payload);
  return data.data;
};

export const updateUser = async (payload) => {
  const { data } = await client.post('/users/update', payload);
  return data.data;
};

export const deleteUser = async (id) => {
  const { data } = await client.post('/users/delete', { id });
  return data.data;
};

export const assignUserRoles = async (payload) => {
  const { data } = await client.post('/users/assign_roles', payload);
  return data.data;
};

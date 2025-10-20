import client from './client';

export const listRoles = async () => {
  const { data } = await client.post('/rbac/role/list', {});
  return data.data;
};

export const createRole = async (payload) => {
  const { data } = await client.post('/rbac/role/create', payload);
  return data.data;
};

export const updateRole = async (payload) => {
  const { data } = await client.post('/rbac/role/update', payload);
  return data.data;
};

export const deleteRole = async (id) => {
  const { data } = await client.post('/rbac/role/delete', { id });
  return data.data;
};

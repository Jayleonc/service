import client from './client';

export const listRoles = async () => {
  const { data } = await client.post('/roles/list', {});
  return data.data;
};

export const createRole = async (payload) => {
  const { data } = await client.post('/roles/create', payload);
  return data.data;
};

export const updateRole = async (payload) => {
  const { data } = await client.post('/roles/update', payload);
  return data.data;
};

export const deleteRole = async (id) => {
  const { data } = await client.post('/roles/delete', { id });
  return data.data;
};

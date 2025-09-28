import axios from 'axios';

let accessToken = '';

export const setAuthToken = (token) => {
  accessToken = token || '';
};

const client = axios.create({
  baseURL: '/v1',
  headers: {
    'Content-Type': 'application/json',
  },
});

client.interceptors.request.use((config) => {
  if (accessToken) {
    config.headers.Authorization = `Bearer ${accessToken}`;
  }
  return config;
});

export default client;

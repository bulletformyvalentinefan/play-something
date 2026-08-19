import { api } from './client'

export const register = (payload) => api.post('/auth/register', payload)

export const login = (payload) => api.post('/auth/login', payload)

export const getUser = (id) => api.get(`/user/${id}`)
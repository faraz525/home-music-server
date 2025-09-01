import axios from 'axios'

export const api = axios.create({ withCredentials: true })

api.interceptors.response.use(
  (res) => res,
  async (error) => {
    if (error.response?.status === 401) {
      try {
        await axios.post('/api/auth/refresh', {}, { withCredentials: true })
        return api.request(error.config)
      } catch (_) {
        // fallthrough
      }
    }
    return Promise.reject(error)
  }
)


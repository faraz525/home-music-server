import axios from 'axios'

function getCookie(name: string) {
  const match = document.cookie.match(new RegExp('(?:^|; )' + name.replace(/([.$?*|{}()\[\]\\\/\+^])/g, '\\$1') + '=([^;]*)'))
  return match ? decodeURIComponent(match[1]) : ''
}

export const api = axios.create({ withCredentials: true })

api.interceptors.request.use((config) => {
  const token = getCookie('access_token')
  if (token) config.headers.Authorization = `Bearer ${token}`
  return config
})

api.interceptors.response.use(
  (res) => res,
  async (error) => {
    if (error.response?.status === 401) {
      try {
        const { data } = await axios.post('/api/auth/refresh', {}, { withCredentials: true })
        if (data?.access_token) document.cookie = `access_token=${encodeURIComponent(data.access_token)}; Path=/; Max-Age=${60 * 15}`
        return api.request(error.config)
      } catch (_) {
        // fallthrough
      }
    }
    return Promise.reject(error)
  }
)


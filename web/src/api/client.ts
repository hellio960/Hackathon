import axios, { AxiosInstance, AxiosRequestConfig, AxiosResponse } from 'axios'
import { message } from 'antd'

class ApiClient {
  private instance: AxiosInstance

  constructor(baseURL: string = '/v1') {
    this.instance = axios.create({
      baseURL,
      timeout: 30000,
      headers: {
        'Content-Type': 'application/json',
      },
    })

    this.instance.interceptors.request.use(
      (config) => {
        return config
      },
      (error) => {
        return Promise.reject(error)
      }
    )

    this.instance.interceptors.response.use(
      (response: AxiosResponse) => {
        const { data } = response
        if (data.code !== undefined && data.code !== 0) {
          message.error(data.msg || '请求失败')
          return Promise.reject(new Error(data.msg || '请求失败'))
        }
        return response
      },
      (error) => {
        if (error.response) {
          const { status, data } = error.response
          message.error(data?.msg || `请求失败: ${status}`)
        } else if (error.request) {
          message.error('网络错误，请检查网络连接')
        } else {
          message.error(error.message || '请求失败')
        }
        return Promise.reject(error)
      }
    )
  }

  async get<T = any>(url: string, config?: AxiosRequestConfig): Promise<T> {
    const response = await this.instance.get(url, config)
    return response.data.data || response.data
  }

  async post<T = any>(url: string, data?: any, config?: AxiosRequestConfig): Promise<T> {
    const response = await this.instance.post(url, data, config)
    return response.data.data || response.data
  }

  async put<T = any>(url: string, data?: any, config?: AxiosRequestConfig): Promise<T> {
    const response = await this.instance.put(url, data, config)
    return response.data.data || response.data
  }

  async delete<T = any>(url: string, config?: AxiosRequestConfig): Promise<T> {
    const response = await this.instance.delete(url, config)
    return response.data.data || response.data
  }
}

export const apiClient = new ApiClient()

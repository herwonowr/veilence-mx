export interface PaginationMeta {
  page: number
  limit: number
  total: number
}

export interface ApiResponse<T> {
  data: T
  error: string | null
  meta?: PaginationMeta
}

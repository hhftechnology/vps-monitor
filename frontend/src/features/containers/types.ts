export interface DockerHost {
  Name: string
  Host: string
}

export interface ContainerInfo {
  id: string
  names: string[]
  image: string
  image_id: string
  command: string
  created: number
  state: string
  status: string
  labels?: Record<string, string>
  host: string
}

export interface ContainersQueryParams {
  search?: string
  state?: string
  sortCreated?: "asc" | "desc"
  groupBy?: "none" | "compose"
  host?: string
}

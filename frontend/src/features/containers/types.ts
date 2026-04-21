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
  historical_stats?: {
    cpu_1h: number
    memory_1h: number
    cpu_12h: number
    memory_12h: number
  }
}

export interface ContainersQueryParams {
  search?: string
  state?: string
  sortCreated?: "asc" | "desc"
  sortBy?: "name" | "state" | "uptime" | "created" | "cpu" | "ram"
  groupBy?: "none" | "compose"
  host?: string
}

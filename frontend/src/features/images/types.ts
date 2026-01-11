export interface ImageInfo {
  id: string;
  repo_tags: string[];
  repo_digests: string[];
  size: number;
  virtual_size: number;
  created: number;
  labels: Record<string, string> | null;
  host: string;
}

export interface ImagePullProgress {
  status: string;
  progress?: string;
  id?: string;
}

export interface ImageRemoveResult {
  untagged: string[];
  deleted: string[];
}

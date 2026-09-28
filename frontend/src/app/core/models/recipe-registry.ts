export interface RegistryRecipeAuthor {
  id: number;
  name: string;
  email?: string;
}

export interface RegistryRecipeCategory {
  id: number;
  name: string;
  description?: string;
}

export interface RegistryRecipeTag {
  id: number;
  name: string;
}

export interface RegistryRecipeVersion {
  revision: number;
  data: Record<string, any>;
  changelog?: string | null;
  created_at: string;
}

export interface RegistryRecipe {
  id: string;
  label: string;
  description?: string | null;
  author: RegistryRecipeAuthor | null;
  category: RegistryRecipeCategory | null;
  tags: RegistryRecipeTag[];
  status: 'pending' | 'approved' | 'rejected';
  created_at: string;
  updated_at: string;
  latest_version: RegistryRecipeVersion | null;
}

export interface RegistryRecipeListResponse {
  count: number;
  next: string | null;
  previous: string | null;
  results: RegistryRecipe[];
}

export interface RegistryRecipeFilterOptions {
  categories: string[];
  authors: string[];
  tags: string[];
}

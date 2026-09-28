import { Injectable, inject } from '@angular/core';
import { Wails, JobChain, ChainStatus, Recipe } from './wails';

@Injectable({
  providedIn: 'root'
})
export class JobChainService {
  private wails = inject(Wails);

  async getAllChains(limit = 100, offset = 0): Promise<JobChain[]> {
    return this.wails.getAllJobChains(limit, offset);
  }

  async getChain(id: string): Promise<JobChain> {
    return this.wails.getJobChain(id);
  }

  async getChainStatus(id: string): Promise<ChainStatus> {
    return this.wails.getJobChainStatus(id);
  }

  async deleteChain(id: string): Promise<void> {
    return this.wails.deleteJobChain(id);
  }

  async saveAsRecipe(chainId: string, label: string, description: string): Promise<Recipe> {
    return this.wails.createRecipeFromChain(chainId, label, description);
  }
}

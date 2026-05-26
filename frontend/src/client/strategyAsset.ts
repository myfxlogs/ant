import { strategyAssetClient } from './connect';
import type { StrategyAsset, StrategyAssetClone } from '../gen/ant/v1/strategy_asset_pb';

export type { StrategyAsset, StrategyAssetClone };

type SubmitAssetReviewParams = {
  sourceTemplateId: string;
  name: string;
  description?: string;
  visibility?: string;
};

export const strategyAssetApi = {
  list: async () => {
    const res = await strategyAssetClient.listStrategyAssets({ limit: 50, offset: 0 });
    return res.assets;
  },
  submitReview: (params: SubmitAssetReviewParams) =>
    strategyAssetClient.submitAssetReview({
      sourceTemplateId: params.sourceTemplateId,
      name: params.name,
      description: params.description ?? '',
      visibility: params.visibility ?? 'private',
    }),
  review: (assetId: string, reviewStatus: string, ratingSummary: string) =>
    strategyAssetClient.reviewStrategyAsset({ assetId, reviewStatus, ratingSummary }),
  clone: (assetId: string, name: string) => strategyAssetClient.cloneStrategyAsset({ assetId, name }),
  listClones: async (assetId: string) => {
    const res = await strategyAssetClient.listAssetClones({ assetId });
    return res.clones;
  },
};

import { strategyRuntimeClient } from './connect';

// ── Strategy version history ────────────────────────────────────────

export const strategyVersionApi = {
  list: async (strategyId: string, limit = 50, offset = 0) => {
    const r = await strategyRuntimeClient.listStrategyVersions({ strategyId, limit, offset });
    return { versions: r.versions, total: r.total };
  },

  get: async (strategyId: string, versionNumber: number) => {
    const r = await strategyRuntimeClient.getStrategyVersion({ strategyId, versionNumber });
    return { version: r.version, sourceCode: r.sourceCode };
  },

  rollback: async (strategyId: string, versionNumber: number) => {
    const r = await strategyRuntimeClient.rollbackStrategyVersion({ strategyId, versionNumber });
    return { newVersion: r.newVersion, restoredSourceCode: r.restoredSourceCode };
  },

  diff: async (strategyId: string, fromVersion: number, toVersion: number) => {
    const r = await strategyRuntimeClient.diffStrategyVersions({ strategyId, fromVersion, toVersion });
    return {
      fromVersion: r.fromVersion, fromSourceCode: r.fromSourceCode,
      toVersion: r.toVersion, toSourceCode: r.toSourceCode,
    };
  },

  updateCode: async (strategyId: string, sourceCode: string, changeSummary: string, compileAudit?: boolean) => {
    const r = await strategyRuntimeClient.updateStrategyCode({
      strategyId, sourceCode, changeSummary, compileAudit: compileAudit || false,
    });
    return {
      newVersion: r.newVersion,
      compileSuccess: r.compileSuccess,
      compileError: r.compileError,
      coverageScore: r.coverageScore,
      blindSpots: r.blindSpots,
    };
  },

  checkCode: async (sourceCode: string) => {
    return await strategyRuntimeClient.checkCode({ sourceCode });
  },
};

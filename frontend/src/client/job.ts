import { jobClient, jobStreamClient } from './connect';
import type { Job, JobEvent } from '../gen/ant/v1/job_pb';

export type { Job, JobEvent };

export const jobApi = {
  get: (jobId: string) => jobClient.getJob({ jobId }),

  cancel: (jobId: string) => jobClient.cancelJob({ jobId }),

  subscribe: async (jobId: string, onEvent: (event: JobEvent) => void, afterSeq = 0) => {
    let lastSeq = afterSeq;
    for await (const event of jobStreamClient.subscribeJob({ jobId, afterSeq })) {
      lastSeq = Math.max(lastSeq, Number(event.seq));
      onEvent(event);
    }
    return lastSeq;
  },
};

import { objectiveScoreClient } from './connect';

export interface ObjectiveKline {
  open_time: string;
  close_time: string;
  open_price: number;
  high_price: number;
  low_price: number;
  close_price: number;
  volume: number;
}

export interface ObjectiveScoreRequest {
  symbol: string;
  timeframe: string;
  klines: ObjectiveKline[];
}

export const strategyServiceApi = {
  postObjectiveScore: async (req: ObjectiveScoreRequest) => {
    const r = await objectiveScoreClient.calculateObjectiveScore({
      symbol: req.symbol,
      timeframe: req.timeframe,
      klines: req.klines.map((k) => ({
        openTime: k.open_time,
        closeTime: k.close_time,
        openPrice: k.open_price,
        highPrice: k.high_price,
        lowPrice: k.low_price,
        closePrice: k.close_price,
        volume: k.volume,
      })),
    });
    return {
      decision: r.decision,
      overall_score: r.overallScore,
      technical_score: r.technicalScore,
      signals: r.signals && {
        rsi: r.signals.rsi,
        macd: r.signals.macd && {
          value: r.signals.macd.value,
          signal_line: r.signals.macd.signalLine,
          histogram: r.signals.macd.histogram,
          signal: r.signals.macd.signal,
          trend: r.signals.macd.trend,
        },
        ma: r.signals.ma,
      },
    };
  },
};

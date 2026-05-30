export interface DefaultTemplateItem {
  id: string;
  nameKey?: string;
  descriptionKey?: string;
  name: string;
  description: string;
  code: string;
  isSystem?: boolean;
  tags?: string[];
  [key: string]: unknown;
}

import default_ma_cross from './templates/default-ma-cross';
import default_rsi_oversold from './templates/default-rsi-oversold';
import default_bollinger_squeeze from './templates/default-bollinger-squeeze';
import default_macd_divergence from './templates/default-macd-divergence';
import default_breakout_volume from './templates/default-breakout-volume';
import default_bb_mean_reversion from './templates/default-bb-mean-reversion';
import default_turtle_trading from './templates/default-turtle-trading';
import default_grid_trading from './templates/default-grid-trading';
import default_dca_buy from './templates/default-dca-buy';
import default_pairs_trading from './templates/default-pairs-trading';
import default_momentum_rotation from './templates/default-momentum-rotation';
import default_test_force_buy from './templates/default-test-force-buy';
import default_rsi from './templates/default-rsi';
import default_macd from './templates/default-macd';

export const DEFAULT_TEMPLATES: DefaultTemplateItem[] = [
  default_ma_cross,
  default_rsi_oversold,
  default_bollinger_squeeze,
  default_macd_divergence,
  default_breakout_volume,
  default_bb_mean_reversion,
  default_turtle_trading,
  default_grid_trading,
  default_dca_buy,
  default_pairs_trading,
  default_momentum_rotation,
  default_test_force_buy,
  default_rsi,
  default_macd,
];

package service

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"go.uber.org/zap"

	"anttrader/internal/ai"
	"anttrader/internal/ai/deepseek"
	"anttrader/internal/ai/openai"
	"anttrader/internal/ai/zhipu"
	"anttrader/pkg/logger"
)

// normalizeLang maps various locale codes to a small set of normalized
// language identifiers used for translation caching and prompt building.
func normalizeLang(lang string) string {
	lang = strings.TrimSpace(strings.ToLower(lang))
	if lang == "" {
		return ""
	}
	if lang == "zh-cn" || lang == "zh_cn" || strings.HasPrefix(lang, "zh-hans") {
		return "zh-cn"
	}
	if lang == "zh-tw" || lang == "zh_tw" || strings.HasPrefix(lang, "zh-hant") || lang == "zh-hk" || lang == "zh-mo" {
		return "zh-tw"
	}
	if strings.HasPrefix(lang, "zh") {
		return "zh-cn"
	}
	if strings.HasPrefix(lang, "ja") {
		return "ja"
	}
	if strings.HasPrefix(lang, "vi") {
		return "vi"
	}
	if strings.HasPrefix(lang, "en") {
		return "en"
	}
	// Fallback: treat unknown codes as English (no translation).
	return "en"
}

// localizeEvents populates LocalizedEvent for each economic event using the
// event_translations table as a cache and Zhipu AI for on-demand translation.
//
// The translation strategy is:
//  1. Build a set of distinct original event titles.
//  2. Load any existing translations for (source, original_text, lang) from
//     event_translations.
//  3. For titles without cached translations, call Zhipu LLM in batch and
//     persist the results.
//  4. Assign translated titles back onto the events slice.
func (s *EconomicCalendarService) localizeEvents(ctx context.Context, events []*EconomicCalendarEvent, lang string) error {
	if s.db == nil {
		return fmt.Errorf("db is not configured for localization")
	}
	if len(events) == 0 {
		return nil
	}

	// Collect unique original titles.
	const source = "FRED_RELEASES_DATES"
	originals := make([]string, 0, len(events))
	seen := make(map[string]struct{})
	for _, e := range events {
		title := strings.TrimSpace(e.Event)
		if title == "" {
			continue
		}
		if _, ok := seen[title]; ok {
			continue
		}
		seen[title] = struct{}{}
		originals = append(originals, title)
	}
	if len(originals) == 0 {
		return nil
	}

	// 1) Load cached translations from DB.
	cached := make(map[string]string, len(originals))
	const selectSQL = `
SELECT translated_text
FROM event_translations
WHERE source = $1 AND original_text = $2 AND lang = $3
LIMIT 1`

	for _, title := range originals {
		var translated string
		if err := s.db.GetContext(ctx, &translated, selectSQL, source, title, lang); err != nil {
			if err == sql.ErrNoRows {
				continue
			}
			// Non-fatal: log and continue.
			logger.Warn("failed to load cached translation", zap.Error(err))
			continue
		}
		cached[title] = strings.TrimSpace(translated)
	}

	// 2) Determine which titles still need translation.
	missing := make([]string, 0, len(originals))
	for _, title := range originals {
		if _, ok := cached[title]; !ok {
			missing = append(missing, title)
		}
	}

	// 3) If there are missing titles, call Zhipu in batch.
	if len(missing) > 0 {
		batchTranslations, err := s.translateBatchWithZhipu(ctx, missing, lang)
		if err != nil {
			// Don't fail the whole request if translation fails; just log.
			logger.Warn("zhipu translation failed", zap.Error(err))
		} else {
			// Persist new translations.
			const insertSQL = `
INSERT INTO event_translations (
    source,
    original_text,
    lang,
    translated_text,
    provider
)
VALUES ($1, $2, $3, $4, $5)
ON CONFLICT (source, original_text, lang) DO UPDATE SET
    translated_text = EXCLUDED.translated_text,
    provider = EXCLUDED.provider,
    updated_at = CURRENT_TIMESTAMP`

			for orig, translated := range batchTranslations {
				translated = strings.TrimSpace(translated)
				if translated == "" {
					continue
				}
				if _, errExec := s.db.ExecContext(ctx, insertSQL, source, orig, lang, translated, "zhipu"); errExec != nil {
					logger.Warn("failed to persist translation", zap.Error(errExec))
					continue
				}
				cached[orig] = translated
			}
		}
	}

	// 4) Assign LocalizedEvent fields where we have translations.
	for _, e := range events {
		title := strings.TrimSpace(e.Event)
		if title == "" {
			continue
		}
		if translated, ok := cached[title]; ok && translated != "" {
			e.LocalizedEvent = translated
		}
	}

	return nil
}

// translateBatchWithZhipu translates a batch of titles into the requested language. It returns a map from original title
// to translated title.
func (s *EconomicCalendarService) translateBatchWithZhipu(ctx context.Context, titles []string, lang string) (map[string]string, error) {
	// 1) Try unified JSON config first: econ.translation.ai_config
	var (
		provider ai.AIProvider
		apiKey   string
		model    string
	)

	type econAIConfig struct {
		Provider string `json:"provider"`
		APIKey   string `json:"api_key"`
		Model    string `json:"model"`
		BaseURL  string `json:"base_url"`
		Enabled  bool   `json:"enabled"`
	}

	if s.dynCfg != nil {
		if raw, enabled, _ := s.dynCfg.GetString(ctx, "econ.translation.ai_config", ""); enabled {
			raw = strings.TrimSpace(raw)
			if raw != "" {
				var cfg econAIConfig
				if err := json.Unmarshal([]byte(raw), &cfg); err == nil && cfg.Enabled {
					p := strings.ToLower(strings.TrimSpace(cfg.Provider))
					apiKey = strings.TrimSpace(cfg.APIKey)
					model = strings.TrimSpace(cfg.Model)
					baseURL := strings.TrimSpace(cfg.BaseURL)
					if apiKey != "" {
						switch p {
						case "zhipu", "glm", "glm-4", "glm-4-flash":
							if model == "" {
								model = "glm-4-flash"
							}
							provider = zhipu.NewClient(apiKey, model)
						case "deepseek":
							if model != "" {
								provider = deepseek.NewClientWithModel(apiKey, model)
							} else {
								provider = deepseek.NewClient(apiKey)
							}
						case "custom", "openai":
							// baseURL 为空时由 openai.NewClient 使用默认值
							provider = openai.NewClient(apiKey, model, baseURL)
						}
					}
				}
			}
		}
	}

	// 2) Backward compatible fallback: legacy econ.translation.zhipu_* or
	// ZHIPU_API_KEY env.
	if provider == nil {
		apiKey = strings.TrimSpace(os.Getenv("ZHIPU_API_KEY"))
		model = "glm-4-flash"
		if s.dynCfg != nil {
			if v, enabled, _ := s.dynCfg.GetString(ctx, "econ.translation.zhipu_api_key", ""); enabled {
				if vv := strings.TrimSpace(v); vv != "" {
					apiKey = vv
				}
			}
			if v, enabled, _ := s.dynCfg.GetString(ctx, "econ.translation.zhipu_model", ""); enabled {
				if vv := strings.TrimSpace(v); vv != "" {
					model = vv
				}
			}
		}
		if apiKey == "" {
			return nil, fmt.Errorf("translation AI is not configured (set econ.translation.ai_config or ZHIPU_API_KEY / econ.translation.zhipu_api_key)")
		}
		provider = zhipu.NewClient(apiKey, model)
	}

	// Map normalized lang code to human-readable language name for the prompt.
	langName := ""
	switch lang {
	case "zh-cn":
		langName = "简体中文"
	case "zh-tw":
		langName = "繁體中文"
	case "ja":
		langName = "日本語"
	case "vi":
		langName = "Tiếng Việt"
	case "en":
		langName = "English"
	default:
		langName = "简体中文"
	}

	systemPrompt := "你是一个专业的金融翻译助手，负责将英文的宏观经济事件名称翻译为目标语言。要求：" +
		"1）只翻译事件标题本身，不添加解释；2）保持机构名、指数名、品牌名等专有名词尽量保留或音译；" +
		"3）严格按照输入顺序逐行输出，每行只包含对应的一条译文；4）不要添加序号、引号或多余文本。" +
		"\n\n目标语言：" + langName

	var sb strings.Builder
	for i, title := range titles {
		title = strings.TrimSpace(title)
		if title == "" {
			continue
		}
		if i > 0 {
			sb.WriteString("\n")
		}
		sb.WriteString(title)
	}
	userPrompt := sb.String()
	if strings.TrimSpace(userPrompt) == "" {
		return map[string]string{}, nil
	}

	resp, err := provider.Chat(ctx, []ai.Message{
		{Role: "system", Content: systemPrompt},
		{Role: "user", Content: userPrompt},
	})
	if err != nil {
		return nil, err
	}

	lines := strings.Split(resp.Content, "\n")
	out := make(map[string]string, len(titles))
	idx := 0
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		if idx >= len(titles) {
			break
		}
		orig := strings.TrimSpace(titles[idx])
		if orig != "" {
			out[orig] = line
		}
		idx++
	}

	return out, nil
}

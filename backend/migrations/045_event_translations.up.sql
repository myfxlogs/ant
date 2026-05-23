-- 045_event_translations.up.sql
-- Store localized translations for external economic event titles (e.g. FRED releases).

CREATE TABLE event_translations (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    source VARCHAR(50) NOT NULL,
    original_text TEXT NOT NULL,
    lang VARCHAR(16) NOT NULL,
    translated_text TEXT NOT NULL,
    provider VARCHAR(50),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    UNIQUE (source, original_text, lang)
);

CREATE INDEX idx_event_translations_source_lang
    ON event_translations(source, lang);

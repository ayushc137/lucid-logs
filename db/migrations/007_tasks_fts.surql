-- Migration: 007_tasks_fts
-- Description: Add full-text search capabilities to tasks table
-- Run: surreal sql --endpoint http://localhost:8000 --username root --password root --namespace daily_journal --database core < db/migrations/007_tasks_fts.surql

-- Create analyzer for task search (class tokenizer, lowercase, ascii, snowball filters)
DEFINE ANALYZER IF NOT EXISTS task_search_analyzer TOKENIZERS class FILTERS lowercase, ascii, snowball(english);

-- Full-text search index for task title
DEFINE INDEX IF NOT EXISTS idx_tasks_title_fts ON TABLE tasks COLUMNS title SEARCH ANALYZER task_search_analyzer BM25 HIGHLIGHTS;

-- Full-text search index for task journal
DEFINE INDEX IF NOT EXISTS idx_tasks_journal_fts ON TABLE tasks COLUMNS journal SEARCH ANALYZER task_search_analyzer BM25 HIGHLIGHTS;

-- Full-text search index for task note
DEFINE INDEX IF NOT EXISTS idx_tasks_note_fts ON TABLE tasks COLUMNS note SEARCH ANALYZER task_search_analyzer BM25 HIGHLIGHTS;

-- Priority-based filtering index
DEFINE INDEX IF NOT EXISTS idx_tasks_user_priority ON TABLE tasks COLUMNS created_by, priority;

-- Date range queries (for filtering by date range)
DEFINE INDEX IF NOT EXISTS idx_tasks_user_end_date ON TABLE tasks COLUMNS created_by, end_date;

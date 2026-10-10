-- FEAT-01 (review 3): every coach, critic and planner prompt was
-- Chinese-only while PRODUCT.md promises English and Simplified Chinese as
-- first-class product languages. The conversation now remembers the locale
-- the client chose when creating it ('en' / 'zh', canonicalized from tags
-- like 'en-US'), and the agent package selects its prompt language from
-- this column. The 'zh' default keeps every existing conversation coaching
-- in the only language it has ever used.
ALTER TABLE insideout.agent_conversations
  ADD COLUMN locale text NOT NULL DEFAULT 'zh';

//! NLP processor for document quality and Jaccard-based similarity.
//! No embeddings; word_tokens used only in memory for batch_similarity.

use std::collections::{HashMap, HashSet};
use unicode_segmentation::UnicodeSegmentation;
use whatlang::detect;

/// Document quality metrics. Only scalar fields are persisted; `word_tokens` is in-memory only.
#[derive(Debug, Clone)]
pub struct DocumentQuality {
    pub word_count: u32,
    pub sentence_count: u32,
    pub reading_time_min: f64,
    pub language: String,
    pub readability_score: f64,
    /// Lowercased word tokens for Jaccard similarity. Not stored in Neo4j.
    pub word_tokens: HashSet<String>,
}

/// Article plus its NLP quality for batch similarity.
#[derive(Debug, Clone)]
pub struct ProcessedArticle {
    pub article_id: String,
    pub quality: DocumentQuality,
}

/// Automated Readability Index: 4.71 * (chars/words) + 0.5 * (words/sentences) - 21.43
fn readability_ari(chars: u32, words: u32, sentences: u32) -> f64 {
    if words == 0 || sentences == 0 {
        return 0.0;
    }
    let chars_f = chars as f64;
    let words_f = words as f64;
    let sentences_f = sentences as f64;
    4.71 * (chars_f / words_f) + 0.5 * (words_f / sentences_f) - 21.43
}

impl Default for NlpProcessor {
    fn default() -> Self {
        Self {}
    }
}

pub struct NlpProcessor;

impl NlpProcessor {
    /// Analyze text and fill DocumentQuality including word_tokens.
    pub fn analyze(&self, text: &str) -> DocumentQuality {
        let words: Vec<&str> = text
            .split_word_bounds()
            .filter(|s| s.chars().any(|c| c.is_alphanumeric()))
            .collect();
        let word_count = words.len() as u32;

        let sentence_count = {
            let segments: Vec<&str> = text
                .split(|c| c == '.' || c == '!' || c == '?')
                .filter(|s| !s.trim().is_empty())
                .collect();
            (segments.len() as u32).max(1)
        };

        let reading_time_min = word_count as f64 / 200.0;

        let language = detect(text)
            .map(|i| i.lang().code().to_string())
            .unwrap_or_else(|| "und".to_string());

        let chars = text.chars().count() as u32;
        let readability_score = readability_ari(chars, word_count, sentence_count);

        let word_tokens: HashSet<String> = words
            .into_iter()
            .map(|s| s.to_lowercase())
            .filter(|s| !s.is_empty())
            .collect();

        DocumentQuality {
            word_count,
            sentence_count,
            reading_time_min,
            language,
            readability_score,
            word_tokens,
        }
    }

    /// Jaccard similarity over word_tokens. Pairs with score >= threshold, at most top_k per node.
    /// Returns (id1, id2, score) with id1 < id2. Empty sets → 0.0, no edge.
    pub fn batch_similarity(
        &self,
        processed: &[ProcessedArticle],
        threshold: f64,
        top_k: usize,
    ) -> Vec<(String, String, f64)> {
        if processed.len() < 2 {
            return vec![];
        }

        let mut scores: Vec<((String, String), f64)> = Vec::new();
        for i in 0..processed.len() {
            for j in (i + 1)..processed.len() {
                let a = &processed[i].quality.word_tokens;
                let b = &processed[j].quality.word_tokens;
                let inter = a.intersection(b).count();
                let union = a.union(b).count();
                let score = if union == 0 { 0.0 } else { inter as f64 / union as f64 };
                if !score.is_nan() && score >= threshold {
                    let id1 = processed[i].article_id.clone();
                    let id2 = processed[j].article_id.clone();
                    let (id1, id2) = if id1 <= id2 { (id1, id2) } else { (id2, id1) };
                    scores.push(((id1, id2), score));
                }
            }
        }

        // For each article, keep at most top_k edges by score desc, then truncate.
        // We have undirected pairs; for each node we count edges (both (a,b) and (b,a) don't exist; we only have (id1,id2) with id1<id2).
        // So we need to group by "article" = either id1 or id2, sort that article's edges by score desc, take top_k.
        let mut by_article: HashMap<String, Vec<((String, String), f64)>> = HashMap::new();
        for (pair, sc) in scores {
            let (ref id1, ref id2) = pair;
            by_article
                .entry(id1.clone())
                .or_default()
                .push((pair.clone(), sc));
            by_article
                .entry(id2.clone())
                .or_default()
                .push((pair, sc));
        }

        let mut seen: HashSet<(String, String)> = HashSet::new();
        let mut out: Vec<(String, String, f64)> = Vec::new();
        for (_id, edges) in by_article.iter_mut() {
            edges.sort_by(|a, b| b.1.partial_cmp(&a.1).unwrap_or(std::cmp::Ordering::Equal));
            for ((id1, id2), score) in edges.iter().take(top_k) {
                let key = (id1.clone(), id2.clone());
                if seen.insert(key) {
                    out.push((id1.clone(), id2.clone(), *score));
                }
            }
        }
        // Re-sort so output is stable (e.g. by id1, id2)
        out.sort_by(|a, b| {
            a.0.cmp(&b.0)
                .then_with(|| a.1.cmp(&b.1))
                .then_with(|| a.2.partial_cmp(&b.2).unwrap_or(std::cmp::Ordering::Equal))
        });
        out
    }
}

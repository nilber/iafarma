-- Script para habilitar a extensão pgvector no PostgreSQL
-- Execute este script no banco de dados iafarma_local

-- Criar a extensão pgvector se ela não existir
CREATE EXTENSION IF NOT EXISTS vector;

-- Adicionar coluna de embedding na tabela products
-- Usando 1536 dimensões que é o padrão do OpenAI text-embedding-ada-002
ALTER TABLE products ADD COLUMN IF NOT EXISTS embedding vector(1536);

-- Criar índice para busca de similaridade
CREATE INDEX IF NOT EXISTS products_embedding_idx ON products USING ivfflat (embedding vector_cosine_ops) WITH (lists = 100);

-- Verificar se a extensão foi instalada
SELECT * FROM pg_extension WHERE extname = 'vector';

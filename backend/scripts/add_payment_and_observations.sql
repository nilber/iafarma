-- Adicionar campos de forma de pagamento e observações ao carrinho
ALTER TABLE carts 
ADD COLUMN IF NOT EXISTS payment_method_id UUID REFERENCES payment_methods(id) ON DELETE SET NULL,
ADD COLUMN IF NOT EXISTS observations TEXT,
ADD COLUMN IF NOT EXISTS change_for VARCHAR(20);

-- Adicionar campos de observações ao pedido
ALTER TABLE orders
ADD COLUMN IF NOT EXISTS observations TEXT,
ADD COLUMN IF NOT EXISTS change_for VARCHAR(20);

-- Criar índice para melhor performance em consultas por forma de pagamento
CREATE INDEX IF NOT EXISTS idx_carts_payment_method_id ON carts(payment_method_id);
